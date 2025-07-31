package v1

import "time"

const (
	ServiceStatusOk     = "ok"
	ServiceStatusDown   = "down"
	ServiceStatusIssues = "issues"
)

type ServiceStatus struct {
	Path       string    `json:"path"`
	LastUpdate time.Time `json:"lastUpdate"`
	Status     string    `json:"status"`
}
