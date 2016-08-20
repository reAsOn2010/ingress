package config

const ()

var ()

// Configuration represents the content of haproxy.cfg file
type Configuration struct {
	EnableSyslog         bool `structs:"enableSyslog,omitempty"`
	HttpRequestTimeout   int  `structs:"httpRequestTimeout"`
	ConnectTimeout       int  `structs:"connectTimeout"`
	ClientTimeout        int  `structs:"clientTimeout"`
	ClientFinTimeout     int  `structs:"clientFinTimeout"`
	ServerTimeout        int  `structs:"serverTimeout"`
	TunnelTimeout        int  `structs:"tunnelTimeout"`
	HttpKeepAliveTimeout int  `structs:"httpKeepAliveTimeout"`
}

// NewDefault returns the default configuration contained
// in the file default-conf.json
func NewDefault() Configuration {
	cfg := Configuration{
		EnableSyslog:         false,
		HttpRequestTimeout:   5,
		ConnectTimeout:       5,
		ClientTimeout:        50,
		ClientFinTimeout:     50,
		ServerTimeout:        50,
		TunnelTimeout:        3600,
		HttpKeepAliveTimeout: 60,
	}

	return cfg
}
