package main

import (
	"flag"
	"fmt"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "/Users/nitesh.sinha/.kube/config",
		"Path to your kubeconfig")
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Printf("Error building from flags: %s \n", err.Error())
		// Running in K8S cluster as a pod. So, read API keys from
		// the serviceaccount files mounted inside the pod
		// by kubelet when that pod is launched
		config, err = rest.InClusterConfig()
		if err != nil {
			fmt.Printf("Error read incluster config: %s", err.Error())
		}
	}
	//fmt.Println(config)
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Error creating client set: %s \n", err.Error())
	}
	ch := make(chan struct{})
	informerFactory := informers.NewSharedInformerFactory(clientset, 10*time.Second)
	ctrl := newController(clientset, informerFactory.Apps().V1().Deployments())
	informerFactory.Start(ch)
	ctrl.run(ch)
}
