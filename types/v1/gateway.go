package v1

type RouterListEntry struct {
	Service  string `json:"service"  validate:"requried"`
	Rule     string `json:"rule"     validate:"required"`
	Provider string `json:"provider" validate:"required"`
}

type Service struct {
	Name               string `json:"name"`
	LoadBalancerConfig struct {
		Servers []struct {
			Url string `json:"url"`
		} `json:"servers"`
	} `json:"loadBalancer"`
	ServerStatus map[string]string `json:"serverStatus"`
}
