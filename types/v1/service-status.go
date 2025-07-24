package v1

import "time"

type ServiceStatus struct {
	ID         string    `json:"status"`
	Name       string    `json:"name"`
	LastUpdate time.Time `json:"lastUpdate"`
}
