package haproxy

type IngressConfig struct {
	Hosts map[string]*Host
}

type Host struct {
	Name     string
	Backends []*Backend
}

type Backend struct {
	HostName            string
	Path                string
	Algorithm           string
	IsDefBackend        bool
	SessionAffinity     bool
	CookieStickySession bool
	Endpoints           []*Endpoint
}

type Endpoint struct {
	Host string
	Port string
}

func NewDefaultEps() []*Endpoint {
	var endpoints []*Endpoint
	defaultEndpoint := Endpoint{Host: "127.0.0.1", Port: "8081"}
	endpoints = append(endpoints, &defaultEndpoint)
	return endpoints
}
