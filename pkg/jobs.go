package pkg

type Source struct {
	Type     string   `json:"type"`
	Cloud    string   `json:"cloud"`
	Interval Duration `json:"interval"`
}
type Destination struct {
	Type       string   `json:"type"`
	Project    string   `json:"project"`
	Zone       string   `json:"zone"`
	ZoneName   string   `json:"zoneName"`
	RecordName string   `json:"recordName"`
	RecordType string   `json:"recordType"`
	IpType     string   `json:"ipType"`
	Ttl        Duration `json:"ttl"`
}

type Filter struct {
	LabelKey   string `json:"labelKey"`
	LabelValue string `json:"labelValue"`
	Type       string `json:"type"`
}
type Job struct {
	Name    string `json:"name"`
	Suspend bool   `json:"suspend"`
	Source  Source `json:"source"`
	TagName string `json:"tagName"`
	// Filters     []Filter    `json:"filters"`
	Destination Destination `json:"destination,omitempty"`
}
type Config struct {
	Jobs []Job `json:"jobs"`
}

type Node struct {
	Name      string `json:"name"`
	PrivateIP string `json:"privateIp"`
	PublicIP  string `json:"publicIp"`
}

type Event struct {
	Nodes []Node `json:"nodes"`
}
