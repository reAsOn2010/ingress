package keepalived

import (
	"sync"
	"text/template"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/util/flowcontrol"
)

const (
	keepalivedConfigFilePath = "/etc/keepalived/keepalived.conf"
)

type Manager struct {
	ConfigFile        string
	reloadRateLimiter flowcontrol.RateLimiter
	reloadLock        *sync.Mutex
	template          *template.Template
}

func NewManager() *Manager {
	keepalived := &Manager{
		ConfigFile:        keepalivedConfigFilePath,
		reloadRateLimiter: flowcontrol.NewTokenBucketRateLimiter(0.1, 1),
		reloadLock:        &sync.Mutex{},
	}

	if err := keepalived.loadTemplate(); err != nil {
		glog.Fatalf("invalid keepalived template: %v.", err)
	}
	return keepalived
}
