package pkg

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
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
		for range ticker.C {
			logger.Info("getting node ips")
			eventChannel <- Event{Nodes: getNodeIps(job)}
		}
	}(job)
	return eventChannel
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

	nodes := make([]Node, 0)

	for _, droplet := range droplets {
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
		nodes = append(nodes, node)
	}
	return nodes
}
