package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/pflag"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/unversioned"
	kubectl_util "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

var (
	flags = pflag.NewFlagSet("", pflag.ExitOnError)

	inCluster = flags.Bool("running-in-cluster", true,
		`Optional, if this controller is running in a kubernetes cluster, use the
		 pod secrets for creating a Kubernetes client.`)
	watchNamespace = flags.String("watch-namespace", api.NamespaceAll,
		`Namespace to watch for Ingress. Default is to watch all namespaces`)
	resyncPeriod = flags.Duration("sync-period", 30*time.Second,
		`Relist and confirm cloud resources this often.`)
	enableKeepalived   = flags.Bool("enable-keepalived", false, `Run keepalived`)
	keepalivedPeerName = flags.String("keepalived-peer-name", "haproxy-ingress", `Keepalived peer name`)
	keepalivedPriority = flags.Int("keepalived-periority", 100, `Keepalived priority`)
	virtualIp          = flags.String("keepalived-vip", "10.0.0.1", `Keepalived virtual ip`)
)

func main() {
	var kubeClient *unversioned.Client
	flags.AddGoFlagSet(flag.CommandLine)
	flags.Parse(os.Args)
	clientConfig := kubectl_util.DefaultClientConfig(flags)

	// workaround of noisy log, see https://github.com/kubernetes/kubernetes/issues/17162
	flag.CommandLine.Parse([]string{})

	var err error
	if *inCluster {
		kubeClient, err = unversioned.NewInCluster()
	} else {
		config, connErr := clientConfig.ClientConfig()
		if connErr != nil {
			glog.Fatalf("error connecting to the client: %v", err)
		}
		kubeClient, err = unversioned.New(config)
	}
	if err != nil {
		glog.Fatalf("failed to create client: %v", err)
	}

	lbc, err := newLoadBalancerController(kubeClient, *resyncPeriod, *watchNamespace, *enableKeepalived, *keepalivedPeerName, *keepalivedPriority, *virtualIp)
	if err != nil {
		glog.Fatalf("%v", err)
	}
	go handleSigterm(lbc)

	lbc.Run()

	for {
		glog.Infof("Handled quit, awaiting pod deletion")
		time.Sleep(30 * time.Second)
	}
}

func handleSigterm(lbc *loadBalancerController) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)
	<-signalChan
	glog.Infof("Received SIGTERM, shutting down")

	exitCode := 0
	if err := lbc.Stop(); err != nil {
		glog.Infof("Error during shutdown %v", err)
		exitCode = 1
	}

	glog.Infof("Exiting with %v", exitCode)
	os.Exit(exitCode)
}
