package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"text/template"

	// "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	// client "k8s.io/kubernetes/pkg/client/unversioned"
	// "k8s.io/kubernetes/pkg/util/flowcontrol"
)

func shellOut(cmd string) {
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to execute %v: %v, err: %v", cmd, string(out), err)
	}
}

type HaproxyIngressData struct {
	StartSyslog string
	extensions.IngressList
}

const (
	haproxyConfigTemplatePath = "./haproxy.tmpl"
	haproxyConfigFilePath     = "/usr/local/etc/haproxy/haproxy.cfg"
	haproxyPidFilePath        = "/var/run/haproxy.pid"
)

func main() {
	// var ingClient client.IngressInterface
	// if kubeClient, err := client.NewInCluster(); err != nil {
	// 	log.Fatalf("Failed to create client: %v.", err)
	// } else {
	// 	ingClient = kubeClient.Extensions().Ingress(api.NamespaceAll)
	// }
	tmpl, err := template.New("haproxy.tmpl").ParseFiles(haproxyConfigTemplatePath)
	if err != nil {
		fmt.Printf("%v", err)
		return
	}
	fmt.Printf("%v", tmpl)
	data := &HaproxyIngressData{
		StartSyslog: "false",
		IngressList: extensions.IngressList{},
	}
	if err := tmpl.Execute(os.Stdout, data); err != nil {
		fmt.Printf("%v", err)
	}
	return
	// rateLimiter := flowcontrol.NewTokenBucketRateLimiter(0.1, 1)
	// known := &extensions.IngressList{}

	// Controller loop
	// shellOut(fmt.Sprintf("haproxy -c -f %s -p %s", haproxyConfigFilePath, haproxyPidFilePath))
	// for {
	// 	rateLimiter.Accept()
	// 	ingresses, err := ingClient.List(api.ListOptions{})
	// 	if err != nil {
	// 		log.Printf("Error retrieving ingresses: %v", err)
	// 		continue
	// 	}
	// 	if reflect.DeepEqual(ingresses.Items, known.Items) {
	// 		continue
	// 	}
	// 	known = ingresses
	// 	if w, err := os.Create(haproxyConfigFilePath); err != nil {
	// 		log.Fatalf("Failed to open %v: %v", haproxyConfigFilePath, err)
	// 	} else if err := tmpl.Execute(w, ingresses); err != nil {
	// 		log.Fatalf("Failed to write template %v", err)
	// 	}
	// 	shellOut(fmt.Sprintf("haproxy -c -f %s -p %s -sf $(cat %s)", haproxyConfigFilePath, haproxyPidFilePath, haproxyPidFilePath))
	// }
}
