package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/reAsOn2010/ingress/controllers/haproxy/haproxy"

	"k8s.io/kubernetes/pkg/api"
	podutil "k8s.io/kubernetes/pkg/api/pod"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/client/record"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/intstr"
	"k8s.io/kubernetes/pkg/watch"
)

const (
	defHostName              = "_"
	podStoreSyncedPollPeriod = 1 * time.Second
	rootLocation             = "/"
	namedPortAnnotation      = "ingress.kubernetes.io/named-ports"
)

var (
	keyFunc = framework.DeletionHandlingMetaNamespaceKeyFunc
)

type namedPortMapping map[string]string

// getPort returns the port defined in a named port
func (npm namedPortMapping) getPort(name string) (string, bool) {
	val, ok := npm.getPortMappings()[name]
	return val, ok
}

// getPortMappings returns the map containing the
// mapping of named port names and the port number
func (npm namedPortMapping) getPortMappings() map[string]string {
	data := npm[namedPortAnnotation]
	var mapping map[string]string
	if data == "" {
		return mapping
	}
	if err := json.Unmarshal([]byte(data), &mapping); err != nil {
		glog.Errorf("unexpected error reading annotations: %v", err)
	}

	return mapping
}

type loadBalancerController struct {
	client         *client.Client
	ingController  *framework.Controller
	svcController  *framework.Controller
	endpController *framework.Controller
	ingLister      StoreToIngressLister
	svcLister      cache.StoreToServiceLister
	endpLister     cache.StoreToEndpointsLister
	haproxy        *haproxy.Manager
	syncQueue      *taskQueue
	ingQueue       *taskQueue

	recorder record.EventRecorder

	stopLock sync.Mutex
	shutdown bool
	stopCh   chan struct{}
}

func newLoadBalancerController(kubeClient *client.Client, resyncPeriod time.Duration, namespace string) (*loadBalancerController, error) {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(kubeClient.Events(namespace))
	lbc := loadBalancerController{
		client:   kubeClient,
		haproxy:  haproxy.NewManager(),
		stopCh:   make(chan struct{}),
		recorder: eventBroadcaster.NewRecorder(api.EventSource{Component: "haproxy-ingress-controller"}),
	}
	lbc.syncQueue = NewTaskQueue(lbc.sync)
	lbc.ingQueue = NewTaskQueue(lbc.updateIngressStatus)
	ingEventHandler := framework.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			addIng := obj.(*extensions.Ingress)
			if !isHaproxyIngress(addIng) {
				glog.Infof("Ignoring add for ingress %v based on annotation %v", addIng.Name, ingressClassKey)
				return
			}
			lbc.recorder.Eventf(addIng, api.EventTypeNormal, "CREATE", fmt.Sprintf("%s/%s", addIng.Namespace, addIng.Name))
			lbc.ingQueue.enqueue(obj)
			lbc.syncQueue.enqueue(obj)
		},
		DeleteFunc: func(obj interface{}) {
			delIng := obj.(*extensions.Ingress)
			if !isHaproxyIngress(delIng) {
				glog.Infof("Ignoring add for ingress %v based on annotation %v", delIng.Name, ingressClassKey)
				return
			}
			lbc.recorder.Eventf(delIng, api.EventTypeNormal, "DELETE", fmt.Sprintf("%s/%s", delIng.Namespace, delIng.Name))
			lbc.syncQueue.enqueue(obj)
		},
		UpdateFunc: func(old, cur interface{}) {
			curIng := cur.(*extensions.Ingress)
			if !isHaproxyIngress(curIng) {
				return
			}
			if !reflect.DeepEqual(old, cur) {
				upIng := cur.(*extensions.Ingress)
				lbc.recorder.Eventf(upIng, api.EventTypeNormal, "UPDATE", fmt.Sprintf("%s/%s", upIng.Namespace, upIng.Name))
				lbc.ingQueue.enqueue(cur)
				lbc.syncQueue.enqueue(cur)
			}
		},
	}

	eventHandler := framework.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			lbc.syncQueue.enqueue(obj)
		},
		DeleteFunc: func(obj interface{}) {
			lbc.syncQueue.enqueue(obj)
		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				lbc.syncQueue.enqueue(cur)
			}
		},
	}

	lbc.ingLister.Store, lbc.ingController = framework.NewInformer(
		&cache.ListWatch{
			ListFunc:  ingressListFunc(lbc.client, namespace),
			WatchFunc: ingressWatchFunc(lbc.client, namespace),
		},
		&extensions.Ingress{}, resyncPeriod, ingEventHandler)
	lbc.svcLister.Store, lbc.svcController = framework.NewInformer(
		&cache.ListWatch{
			ListFunc:  serviceListFunc(lbc.client, namespace),
			WatchFunc: serviceWatchFunc(lbc.client, namespace),
		},
		&api.Service{}, resyncPeriod, framework.ResourceEventHandlerFuncs{})
	lbc.endpLister.Store, lbc.endpController = framework.NewInformer(
		&cache.ListWatch{
			ListFunc:  endpointsListFunc(lbc.client, namespace),
			WatchFunc: endpointsWatchFunc(lbc.client, namespace),
		},
		&api.Endpoints{}, resyncPeriod, eventHandler)
	return &lbc, nil
}

func ingressListFunc(c *client.Client, ns string) func(api.ListOptions) (runtime.Object, error) {
	return func(opts api.ListOptions) (runtime.Object, error) {
		return c.Extensions().Ingress(ns).List(opts)
	}
}

func ingressWatchFunc(c *client.Client, ns string) func(options api.ListOptions) (watch.Interface, error) {
	return func(options api.ListOptions) (watch.Interface, error) {
		return c.Extensions().Ingress(ns).Watch(options)
	}
}

func serviceListFunc(c *client.Client, ns string) func(api.ListOptions) (runtime.Object, error) {
	return func(opts api.ListOptions) (runtime.Object, error) {
		return c.Services(ns).List(opts)
	}
}

func serviceWatchFunc(c *client.Client, ns string) func(options api.ListOptions) (watch.Interface, error) {
	return func(options api.ListOptions) (watch.Interface, error) {
		return c.Services(ns).Watch(options)
	}
}

func endpointsListFunc(c *client.Client, ns string) func(api.ListOptions) (runtime.Object, error) {
	return func(opts api.ListOptions) (runtime.Object, error) {
		return c.Endpoints(ns).List(opts)
	}
}

func endpointsWatchFunc(c *client.Client, ns string) func(options api.ListOptions) (watch.Interface, error) {
	return func(options api.ListOptions) (watch.Interface, error) {
		return c.Endpoints(ns).Watch(options)
	}
}

func (lbc *loadBalancerController) sync(key string) error {
	if !lbc.controllersInSync() {
		time.Sleep(podStoreSyncedPollPeriod)
		return fmt.Errorf("deferring sync till endpoints controller has synced")
	}
	ings := lbc.ingLister.Store.List()
	haConfig := lbc.haproxy.ReadConfig()
	hosts := lbc.getHosts(ings)
	return lbc.haproxy.CheckAndReload(haConfig, haproxy.IngressConfig{
		Hosts: hosts,
		// TODO: add layer 4
	})
}

func (lbc *loadBalancerController) getSvc(name string) (*api.Service, error) {
	svcObj, svcExists, err := lbc.svcLister.Store.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !svcExists {
		return nil, fmt.Errorf("Cannot find svc: %v", name)
	}
	return svcObj.(*api.Service), nil
}

func (lbc *loadBalancerController) getHosts(data []interface{}) map[string]*haproxy.Host {
	hosts := lbc.createHosts(data)
	for _, ingIf := range data {
		ing := ingIf.(*extensions.Ingress)

		for _, rule := range ing.Spec.Rules {
			if rule.IngressRuleValue.HTTP == nil {
				continue
			}
			hostname := rule.Host
			if hostname == "" {
				hostname = defHostName
			}
			host := hosts[hostname]
			if host == nil {
				host = hosts["_"]
			}
			for _, path := range rule.HTTP.Paths {
				haPath := path.Path
				// if there's no path defined we assume /
				if haPath == "" {
					lbc.recorder.Eventf(ing, api.EventTypeWarning, "MAPPING",
						"Ingress rule '%v/%v' contains no path definition. Assuming /",
						ing.GetNamespace(), ing.GetName())
					haPath = rootLocation
				}
				addBackend := true
				for _, backend := range host.Backends {
					if backend.Path == rootLocation && haPath == rootLocation && backend.IsDefBackend {
						if svc, err := lbc.getSvc(fmt.Sprintf("%s/%s", ing.GetNamespace(), path.Backend.ServiceName)); err != nil {
							lbc.recorder.Eventf(ing, api.EventTypeWarning, "MAPPING",
								"Fetch service failed of '%v/%v'", ing.GetNamespace(), path.Backend.ServiceName)
						} else {
							backend.Endpoints = lbc.getEndpoints(svc, path.Backend.ServicePort, api.ProtocolTCP)
							backend.HostName = hostname
							backend.Algorithm = "leastconn"
							backend.SessionAffinity = false
							backend.CookieStickySession = false
						}
						addBackend = false
						continue
					}

					if backend.Path == haPath {
						lbc.recorder.Eventf(ing, api.EventTypeWarning, "MAPPING",
							"Path '%v' already defined in another Ingress rule", haPath)
						addBackend = false
						break
					}
				}
				if addBackend {
					if svc, err := lbc.getSvc(fmt.Sprintf("%s/%s", ing.GetNamespace(), path.Backend.ServiceName)); err != nil {
						lbc.recorder.Eventf(ing, api.EventTypeWarning, "MAPPING",
							"Fetch service failed of '%v/%v'", ing.GetNamespace(), path.Backend.ServiceName)
					} else {
						host.Backends = append(host.Backends, &haproxy.Backend{
							Path:                haPath,
							Endpoints:           lbc.getEndpoints(svc, path.Backend.ServicePort, api.ProtocolTCP),
							HostName:            hostname,
							Algorithm:           "leastconn",
							SessionAffinity:     false,
							CookieStickySession: false,
						})
					}
				}
			}
		}
	}
	return hosts
}

func (lbc *loadBalancerController) createHosts(data []interface{}) map[string]*haproxy.Host {
	hosts := make(map[string]*haproxy.Host)

	for _, ingIf := range data {
		ing := ingIf.(*extensions.Ingress)

		for _, rule := range ing.Spec.Rules {
			hostname := rule.Host
			if hostname == "" {
				hostname = defHostName
			}
			if _, ok := hosts[hostname]; !ok {
				var backends []*haproxy.Backend
				backends = append(backends, &haproxy.Backend{
					HostName:     hostname,
					Path:         rootLocation,
					IsDefBackend: true,
					Endpoints:    haproxy.NewDefaultEps(),
				})
				hosts[hostname] = &haproxy.Host{Name: hostname, Backends: backends}
			}
		}
	}
	return hosts
}

func (lbc *loadBalancerController) getEndpoints(s *api.Service, servicePort intstr.IntOrString, proto api.Protocol) []*haproxy.Endpoint {
	glog.V(3).Infof("getting endpoints for service %v/%v and port %v", s.Namespace, s.Name, servicePort.String())
	ep, err := lbc.endpLister.GetServiceEndpoints(s)
	if err != nil {
		glog.Warningf("unexpected error obtaining service endpoints: %v", err)
		return []*haproxy.Endpoint{}
	}

	endpoints := []*haproxy.Endpoint{}

	for _, ss := range ep.Subsets {
		for _, epPort := range ss.Ports {

			if !reflect.DeepEqual(epPort.Protocol, proto) {
				continue
			}

			var targetPort int32
			switch servicePort.Type {
			case intstr.Int:
				if int(epPort.Port) == servicePort.IntValue() {
					targetPort = epPort.Port
				}
			case intstr.String:
				namedPorts := s.ObjectMeta.Annotations
				val, ok := namedPortMapping(namedPorts).getPort(servicePort.StrVal)
				if ok {
					port, err := strconv.Atoi(val)
					if err != nil {
						glog.Warningf("%v is not valid as a port", val)
						continue
					}

					targetPort = int32(port)
				} else {
					newnp, err := lbc.checkSvcForUpdate(s)
					if err != nil {
						glog.Warningf("error mapping service ports: %v", err)
						continue
					}
					val, ok := namedPortMapping(newnp).getPort(servicePort.StrVal)
					if ok {
						port, err := strconv.Atoi(val)
						if err != nil {
							glog.Warningf("%v is not valid as a port", val)
							continue
						}

						targetPort = int32(port)
					}
				}
			}

			if targetPort == 0 {
				continue
			}

			for _, epAddress := range ss.Addresses {
				endpoint := haproxy.Endpoint{
					Host: epAddress.IP,
					Port: fmt.Sprintf("%v", targetPort),
				}
				endpoints = append(endpoints, &endpoint)
			}
		}
	}

	glog.V(3).Infof("endpoints found: %v", endpoints)
	return endpoints
}

// checkSvcForUpdate verifies if one of the running pods for a service contains
// named port. If the annotation in the service does not exists or is not equals
// to the port mapping obtained from the pod the service must be updated to reflect
// the current state
func (lbc *loadBalancerController) checkSvcForUpdate(svc *api.Service) (map[string]string, error) {
	// get the pods associated with the service
	// TODO: switch this to a watch
	pods, err := lbc.client.Pods(svc.Namespace).List(api.ListOptions{
		LabelSelector: labels.Set(svc.Spec.Selector).AsSelector(),
	})

	namedPorts := map[string]string{}
	if err != nil {
		return namedPorts, fmt.Errorf("error searching service pods %v/%v: %v", svc.Namespace, svc.Name, err)
	}

	if len(pods.Items) == 0 {
		return namedPorts, nil
	}

	// we need to check only one pod searching for named ports
	pod := &pods.Items[0]
	glog.V(4).Infof("checking pod %v/%v for named port information", pod.Namespace, pod.Name)
	for i := range svc.Spec.Ports {
		servicePort := &svc.Spec.Ports[i]

		_, err := strconv.Atoi(servicePort.TargetPort.StrVal)
		if err != nil {
			portNum, err := podutil.FindPort(pod, servicePort)
			if err != nil {
				glog.V(4).Infof("failed to find port for service %s/%s: %v", svc.Namespace, svc.Name, err)
				continue
			}

			if servicePort.TargetPort.StrVal == "" {
				continue
			}

			namedPorts[servicePort.TargetPort.StrVal] = fmt.Sprintf("%v", portNum)
		}
	}

	if svc.ObjectMeta.Annotations == nil {
		svc.ObjectMeta.Annotations = map[string]string{}
	}

	curNamedPort := svc.ObjectMeta.Annotations[namedPortAnnotation]
	if len(namedPorts) > 0 && !reflect.DeepEqual(curNamedPort, namedPorts) {
		data, _ := json.Marshal(namedPorts)

		newSvc, err := lbc.client.Services(svc.Namespace).Get(svc.Name)
		if err != nil {
			return namedPorts, fmt.Errorf("error getting service %v/%v: %v", svc.Namespace, svc.Name, err)
		}

		if newSvc.ObjectMeta.Annotations == nil {
			newSvc.ObjectMeta.Annotations = map[string]string{}
		}

		newSvc.ObjectMeta.Annotations[namedPortAnnotation] = string(data)
		glog.Infof("updating service %v with new named port mappings", svc.Name)
		_, err = lbc.client.Services(svc.Namespace).Update(newSvc)
		if err != nil {
			return namedPorts, fmt.Errorf("error syncing service %v/%v: %v", svc.Namespace, svc.Name, err)
		}

		return newSvc.ObjectMeta.Annotations, nil
	}

	return namedPorts, nil
}

// TODO: implement this
func (lbc *loadBalancerController) updateIngressStatus(key string) error {
	glog.Infof("update")
	return nil
}

func (lbc *loadBalancerController) controllersInSync() bool {
	return lbc.ingController.HasSynced() &&
		lbc.svcController.HasSynced() &&
		lbc.endpController.HasSynced()
}

// Stop stops the loadbalancer controller.
func (lbc *loadBalancerController) Stop() error {
	// Stop is invoked from the http endpoint.
	lbc.stopLock.Lock()
	defer lbc.stopLock.Unlock()

	// Only try draining the workqueue if we haven't already.
	if !lbc.shutdown {
		lbc.shutdown = true
		close(lbc.stopCh)

		// ings := lbc.ingLister.Store.List()
		// glog.Infof("removing IP address %v from ingress rules", lbc.podInfo.NodeIP)
		// lbc.removeFromIngress(ings)

		glog.Infof("Shutting down controller queues.")
		lbc.syncQueue.shutdown()
		lbc.ingQueue.shutdown()

		return nil
	}

	return fmt.Errorf("shutdown already in progress")
}

// Run starts the loadbalancer controller.
func (lbc *loadBalancerController) Run() {
	glog.Infof("starting haproxy loadbalancer controller")
	go lbc.haproxy.Start()
	go lbc.ingController.Run(lbc.stopCh)
	go lbc.svcController.Run(lbc.stopCh)
	go lbc.endpController.Run(lbc.stopCh)

	go lbc.syncQueue.run(time.Second, lbc.stopCh)
	go lbc.ingQueue.run(time.Second, lbc.stopCh)

	<-lbc.stopCh
}
