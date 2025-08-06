package traefik

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	config "microservice/internal/configuration"
	v1 "microservice/types/v1"
)

const (
	fPathPrefixRule = "PathPrefix(`%s`)"
	fPathRule       = "Path(`%s`)"
)

func ServiceStatus(paths ...string) (statuses []v1.ServiceStatus, err error) {
	c := config.Default.Viper()
	baseUrl := c.GetString(config.ConfigurationKey_TraefikAPIEndpoint)
	routerOverview, err := url.JoinPath(baseUrl, "/api", "/http", "/routers")
	if err != nil {
		return nil, err
	}
	res, err := http.Get(routerOverview)
	if err != nil {
		return nil, err
	}

	var Routers []v1.RouterListEntry
	err = json.NewDecoder(res.Body).Decode(&Routers)
	if err != nil {
		return nil, err
	}

	observedRouters := make(map[string]v1.RouterListEntry)

	for _, path := range paths {
		var expectedRules []string
		expectedRules = append(expectedRules, fmt.Sprintf(fPathPrefixRule, path))
		expectedRules = append(expectedRules, fmt.Sprintf(fPathRule, path))

		for _, router := range Routers {
			if router.Provider != "docker" {
				continue
			}

			for _, rule := range expectedRules {
				if !strings.Contains(router.Rule, rule) {
					continue
				}

				observedRouters[path] = router
			}

		}
	}

	for path, router := range observedRouters {
		serviceName := fmt.Sprintf("%s@%s", router.Service, router.Provider)

		serviceDetailUrl, err := url.JoinPath(baseUrl, "/api/http/services/", serviceName)
		if err != nil {
			return nil, err
		}

		detailResponse, err := http.Get(serviceDetailUrl)
		if err != nil {
			return nil, err
		}

		var service v1.Service
		err = json.NewDecoder(detailResponse.Body).Decode(&service)
		if err != nil {
			return nil, err
		}

		var upstreamStatuses []bool

		for _, upstream := range service.LoadBalancerConfig.Servers {
			upstreamStatus, ok := service.ServerStatus[upstream.Url]
			if !ok {
				upstreamStatuses = append(upstreamStatuses, false)
				continue
			}

			if upstreamStatus != "UP" {
				upstreamStatuses = append(upstreamStatuses, false)
				continue
			}

			upstreamStatuses = append(upstreamStatuses, true)
		}

		status := v1.ServiceStatus{
			Path:       path,
			LastUpdate: time.Now(),
			Status:     v1.ServiceStatusDown,
		}

		for _, upstreamAvailable := range upstreamStatuses {
			if upstreamAvailable {
				status.Status = v1.ServiceStatusOk
			} else {
				status.Status = v1.ServiceStatusDown
				break
			}
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}
