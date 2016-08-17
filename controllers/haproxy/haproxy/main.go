package haproxy

import (
	"sync"
	"text/template"

	"github.com/golang/glog"
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
	reloadRateLimiter flowcontrol.RateLimiter
	reloadLock        *sync.Mutex
	template          *template.Template
}

func NewManager() *Manager {
	haproxy := &Manager{
		BinWrapper:        haproxyWrapper,
		PidFile:           haproxyPidFilePath,
		ConfigFile:        haproxyConfigFilePath,
		reloadRateLimiter: flowcontrol.NewTokenBucketRateLimiter(0.1, 1),
		reloadLock:        &sync.Mutex{},
	}

	if err := haproxy.loadTemplate(); err != nil {
		glog.Fatalf("invalid haproxy template: %v.", err)
	}
	return haproxy
}
