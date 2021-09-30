package pkg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/digitalocean/godo"
	log "github.com/sirupsen/logrus"
	dns "google.golang.org/api/dns/v1beta2"
)

var logger = log.Logger{
	Out: os.Stdout,
	Formatter: &log.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.000",
	},
	Level: log.InfoLevel,
}

var (
	lastUpdatedEventByRecord = make(map[string]Event)
)

//@todo: refactor with readable logs

type Duration struct {
	time.Duration
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value)
		return nil
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil
	default:
		return errors.New("invalid duration")
	}
}

func updateRecord(event Event, job Job) bool {
	logger.Info("updating record")
	// @todo: option to use multi ip A record option, instead of weightage of ips

	items := make([]*dns.RRSetRoutingPolicyWrrPolicyWrrPolicyItem, 0)

	weightage := 1000 / len(event.Nodes)
	totalWeightage := 1000

	logger.Info("weightage: ", weightage)

	for _, node := range event.Nodes {
		if totalWeightage < 2*weightage {
			weightage = totalWeightage
		}
		ip := node.PrivateIP
		if job.Destination.IpType == "public" {
			ip = node.PublicIP
		}
		item := dns.RRSetRoutingPolicyWrrPolicyWrrPolicyItem{
			Weight: float64(weightage),
			Rrdatas: []string{
				ip,
			},
		}
		items = append(items, &item)
		logger.Info("weightage: ", weightage)
		totalWeightage -= weightage
	}

	// @todo: add actionable logs
	logger.Info("items: ", items)

	ctx := context.Background()
	dnsService, err := dns.NewService(ctx)
	if err != nil {
		logger.Fatalln(err)
	}
	rrss := dns.NewResourceRecordSetsService(dnsService)
	//@todo: create record if not exists, with config flag
	call := rrss.Patch(job.Destination.Project, job.Destination.ZoneName, job.Destination.RecordName, job.Destination.RecordType, &dns.ResourceRecordSet{
		Name: job.Destination.RecordName,
		Type: job.Destination.RecordType,
		Ttl:  int64(job.Destination.Ttl.Seconds()),
		RoutingPolicy: &dns.RRSetRoutingPolicy{
			Wrr: &dns.RRSetRoutingPolicyWrrPolicy{
				Items: items,
			},
		},
	})
	rs, err := call.Do()
	if err != nil {
		logger.Fatalln(err)
	}
	logger.Info("updated record: ", rs)
	return true
}
func syncZone(event Event, job Job) {
	logger.Info("syncing zone")
	logger.Info("event: ", event)

	if lastEvent, ok := lastUpdatedEventByRecord[job.Destination.RecordName]; ok {
		if reflect.DeepEqual(lastEvent, event) {
			logger.Info("no change")
			return
		}
	}
	if updateRecord(event, job) {
		lastUpdatedEventByRecord[job.Destination.RecordName] = event
	}

}
func getNodeIPChangeEvents(wg *sync.WaitGroup, job Job) chan Event {
	eventChannel := make(chan Event)
	wg.Add(1)
	go func(job Job) {
		defer wg.Done()
		ticker := time.NewTicker(job.Source.Interval.Duration)
		for ; true; <-ticker.C {
			logger.Info("getting node ips")
			nodes := getNodeIps(job)
			if len(nodes) > 0 {
				eventChannel <- Event{Nodes: nodes}
			}
		}
	}(job)
	return eventChannel
}
func FileExists(name string) (bool, error) {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func getNodeIps(job Job) []Node {
	token := os.Getenv("DIGITALOCEAN_TOKEN")

	client := godo.NewFromToken(token)
	ctx := context.TODO()

	//@todo: do droplet pagination should be handled
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	droplets, _, err := client.Droplets.ListByTag(ctx, job.TagName, opt)
	if err != nil {
		//@todo: better error messages
		logger.Fatalln(err)
	}

	logger.Info("total droplet count: ", len(droplets))

	nodes := make([]Node, 0)

	mutex := &sync.Mutex{}
	wg := &sync.WaitGroup{}

	var ignoredNodes IgnoredNodes
	if len(job.Destination.IgnoreListFilePath) > 0 {
		if ok, err := FileExists(job.Destination.IgnoreListFilePath); ok && (err == nil) {
			logger.Info("loading ignore list config")
			if fileContent, err := ioutil.ReadFile(job.Destination.IgnoreListFilePath); err == nil {
				err := json.Unmarshal(fileContent, &ignoredNodes)
				if err != nil {
					logger.Fatalln(err)
				}
			}
		}
	}

	for _, droplet := range droplets {
		// @todo: filter nodes from redis ignore list by key `job:node_name` if present
		ignored := false
		for _, ignoredNode := range ignoredNodes.Nodes {
			key := fmt.Sprintf("%s:%s", job.Name, droplet.Name)
			if strings.EqualFold(key, ignoredNode.Name) {
				logger.Info("ignored node found: ", key, " --> ", ignoredNode.Name)
				ignored = true
			}
		}
		if ignored {
			continue
		}

		wg.Add(1)
		go func(droplet godo.Droplet) {
			defer wg.Done()
			node := Node{}
			for _, ip := range droplet.Networks.V4 {
				if ip.Type == "private" {
					node.PrivateIP = ip.IPAddress
				}
				if ip.Type == "public" {
					node.PublicIP = ip.IPAddress
				}
			}
			node.Name = droplet.Name
			isReady := false
			successCount := 0
			failureCount := 0
			// @todo: handle or throw error on misconfiguration of interval and threshold
			started := time.Now()
			healthCheckNotRequired := true
			if len(job.Destination.ReadinessProbe.HTTPGet.Path) > 0 {
				healthCheckNotRequired = false
				protocal := "http"
				if job.Destination.ReadinessProbe.HTTPGet.Scheme == "HTTP" {
					protocal = "http"
				} else {
					protocal = "https"
				}
				readinessProbeEndpoint := fmt.Sprintf(
					"%s://%s:%d%s",
					protocal,
					node.PublicIP,
					job.Destination.ReadinessProbe.HTTPGet.Port,
					job.Destination.ReadinessProbe.HTTPGet.Path,
				)
				ticker := time.NewTicker(job.Destination.ReadinessProbe.Period.Duration)
				for ; true; <-ticker.C {
					if failureCount >= job.Destination.ReadinessProbe.FailureThreshold {
						ticker.Stop()
						logger.Info("node is not ready: ", node, ", failureThreshold reached")
						break
					}
					logger.Info("checking readiness: ", readinessProbeEndpoint)
					resp, err := http.Get(readinessProbeEndpoint)
					if err != nil {
						logger.Error(err)
						failureCount++
						continue
					}
					if resp.StatusCode == 200 {
						successCount++
					} else {
						failureCount++
					}
					if successCount >= job.Destination.ReadinessProbe.SuccessThreshold {
						isReady = true
						ticker.Stop()
						logger.Info("node is ready: ", node)
						break
					} else {
						if time.Since(started).Seconds() > job.Destination.ReadinessProbe.ProgressDeadline.Seconds() {
							ticker.Stop()
							logger.Info("node is not ready: ", node, ", progressDeadline reached")
							break
						}
					}
				}
			}
			if isReady || healthCheckNotRequired {
				mutex.Lock()
				nodes = append(nodes, node)
				mutex.Unlock()
			}
		}(droplet)
	}
	wg.Wait()
	return nodes
}
