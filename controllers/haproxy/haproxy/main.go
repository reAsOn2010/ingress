package haproxy

import (
	"sync"
	"text/template"

	"github.com/golang/glog"
	"github.com/reAsOn2010/ingress/controllers/haproxy/haproxy/config"

	"k8s.io/kubernetes/pkg/util/flowcontrol"
)

const (
	haproxyWrapper        = "/usr/local/sbin/haproxy-systemd-wrapper"
	haproxyConfigFilePath = "/usr/local/etc/haproxy/haproxy.cfg"
	haproxyPidFilePath    = "/var/run/haproxy.pid"
)

type Manager struct {
	BinWrapper        string
	ConfigFile        string
	PidFile           string
	sslDHParam        string
	reloadRateLimiter flowcontrol.RateLimiter
	template          *template.Template
	reloadLock        *sync.Mutex
}

func NewManager() *Manager {
	haproxy := &Manager{
		BinWrapper:        haproxyWrapper,
		PidFile:           haproxyPidFilePath,
		ConfigFile:        haproxyConfigFilePath,
		reloadRateLimiter: flowcontrol.NewTokenBucketRateLimiter(0.1, 1),
		reloadLock:        &sync.Mutex{},
	}
	haproxy.sslDHParam = haproxy.SearchDHParamFile(config.SSLDirectory)
	if err := haproxy.loadTemplate(); err != nil {
		glog.Fatalf("invalid haproxy template: %v.", err)
	}
	return haproxy
}
