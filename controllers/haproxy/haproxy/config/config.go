package config

const ()

var ()

// Configuration represents the content of haproxy.cfg file
type Configuration struct {
	EnableSyslog bool `structs:"enableSyslog,omitempty"`
}

// NewDefault returns the default configuration contained
// in the file default-conf.json
func NewDefault() Configuration {
	cfg := Configuration{
		EnableSyslog: false,
	}

	return cfg
}
