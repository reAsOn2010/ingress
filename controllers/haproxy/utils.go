package main

import (
	"io/ioutil"
	"strings"
	"time"

	"github.com/golang/glog"

	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/util/wait"
	"k8s.io/kubernetes/pkg/util/workqueue"
)

// StoreToIngressLister makes a Store that lists Ingress.
type StoreToIngressLister struct {
	cache.Store
}

type StoreToSecretsLister struct {
	cache.Store
}

type ingAnnotations map[string]string

func (ing ingAnnotations) ingressClass() string {
	val, ok := ing[ingressClassKey]
	if !ok {
		return ""
	}
	return val
}

const (
	// ingressClassKey picks a specific "class" for the Ingress. The controller
	// only processes Ingresses with this annotation.
	ingressClassKey     = "kubernetes.io/ingress.class"
	haproxyIngressClass = "haproxy"
)

// isHaproxyIngress returns true if the given Ingress either doesn't specify the
// ingress.class annotation, or it's set to "haproxy".
func isHaproxyIngress(ing *extensions.Ingress) bool {
	class := ingAnnotations(ing.ObjectMeta.Annotations).ingressClass()
	return class == haproxyIngressClass
}

// taskQueue manages a work queue through an independent worker that
// invokes the given sync function for every work item inserted.
type taskQueue struct {
	// queue is the work queue the worker polls
	queue workqueue.RateLimitingInterface
	// sync is called for each item in the queue
	sync func(string) error
	// workerDone is closed when the worker exits
	workerDone chan struct{}
}

func (t *taskQueue) run(period time.Duration, stopCh <-chan struct{}) {
	wait.Until(t.worker, period, stopCh)
}

// enqueue enqueues ns/name of the given api object in the task queue.
func (t *taskQueue) enqueue(obj interface{}) {
	key, err := keyFunc(obj)
	if err != nil {
		glog.Infof("could not get key for object %+v: %v", obj, err)
		return
	}
	t.queue.Add(key)
}

func (t *taskQueue) requeue(key string) {
	t.queue.AddRateLimited(key)
}

// worker processes work in the queue through sync.
func (t *taskQueue) worker() {
	for {
		key, quit := t.queue.Get()
		if quit {
			close(t.workerDone)
			return
		}
		glog.V(3).Infof("syncing %v", key)
		if err := t.sync(key.(string)); err != nil {
			glog.Warningf("requeuing %v, err %v", key, err)
			t.requeue(key.(string))
		} else {
			t.queue.Forget(key)
		}

		t.queue.Done(key)
	}
}

// shutdown shuts down the work queue and waits for the worker to ACK
func (t *taskQueue) shutdown() {
	t.queue.ShutDown()
	<-t.workerDone
}

// NewTaskQueue creates a new task queue with the given sync function.
// The sync function is called for every element inserted into the queue.
func NewTaskQueue(syncFn func(string) error) *taskQueue {
	return &taskQueue{
		queue:      workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		sync:       syncFn,
		workerDone: make(chan struct{}),
	}
}

const (
	snakeOilPem = "/etc/ssl/certs/ssl-cert-snakeoil.pem"
	snakeOilKey = "/etc/ssl/private/ssl-cert-snakeoil.key"
)

// getFakeSSLCert returns the snake oil ssl certificate created by the command
// make-ssl-cert generate-default-snakeoil --force-overwrite
func getFakeSSLCert() (string, string) {
	cert, err := ioutil.ReadFile(snakeOilPem)
	if err != nil {
		return "", ""
	}

	key, err := ioutil.ReadFile(snakeOilKey)
	if err != nil {
		return "", ""
	}

	return string(cert), string(key)
}

func isHostValid(host string, cns []string) bool {
	for _, cn := range cns {
		if matchHostnames(cn, host) {
			return true
		}
	}

	return false
}

func matchHostnames(pattern, host string) bool {
	host = strings.TrimSuffix(host, ".")
	pattern = strings.TrimSuffix(pattern, ".")

	if len(pattern) == 0 || len(host) == 0 {
		return false
	}

	patternParts := strings.Split(pattern, ".")
	hostParts := strings.Split(host, ".")

	if len(patternParts) != len(hostParts) {
		return false
	}

	for i, patternPart := range patternParts {
		if i == 0 && patternPart == "*" {
			continue
		}
		if patternPart != hostParts[i] {
			return false
		}
	}

	return true
}
