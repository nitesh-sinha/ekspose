package main

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	appsv1informers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	appsv1listers "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type controller struct {
	clientset      kubernetes.Interface           // to interact with K8s cluster
	depLister      appsv1listers.DeploymentLister // to list deployments
	depCacheSynced cache.InformerSynced           // to check if Informer cache has been sync'ed or not
	queue          workqueue.RateLimitingInterface
}

func newController(clientset kubernetes.Interface,
	depInformer appsv1informers.DeploymentInformer) *controller {
	c := &controller{
		clientset:      clientset,
		depLister:      depInformer.Lister(),
		depCacheSynced: depInformer.Informer().HasSynced,
		queue: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			"ekspose"),
	}
	depInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.handleDeploymentAdd,
			DeleteFunc: c.handleDeploymentDelete,
		},
	)

	return c
}

func (c *controller) run(ch <-chan struct{}) {
	fmt.Println("Starting the custom controller!")
	if !cache.WaitForCacheSync(ch, c.depCacheSynced) {
		fmt.Println("Waiting for the cache to be synced")
	}
	go wait.Until(c.worker, 1*time.Second, ch)
	<-ch // Read from channel to block the controller from stopping
}

func (c *controller) worker() {
	for c.processItem() {

	}
}

func (c *controller) processItem() bool {
	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	// Delete item from queue upon no error in processing it
	defer c.queue.Forget(item)
	key, err := cache.MetaNamespaceKeyFunc(item)
	if err != nil {
		fmt.Println("Error fetching key from Informer cache: ", err.Error())
		return false
	}
	ns, depName, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		fmt.Println("Error splitting key into ns and deployment name: ", err.Error())
		return false
	}
	err = c.processDeployment(ns, depName)
	if err != nil {
		// retry(i.e. add the obj back to queue)
		fmt.Println("Error processing deployment object: ", err.Error())
		return false
	}
	return true
}

func (c *controller) processDeployment(ns, depName string) error {
	// Get the deployment from the lister
	fmt.Println("Dep name: ", depName)
	dep, err := c.depLister.Deployments(ns).Get(depName)
	if err != nil {
		fmt.Println("Error getting deployment from lister: ", err.Error())
		return err
	}
	fmt.Printf("creating service with name: %s in ns: %s \n", dep.Name, ns)
	// create the corresponding service
	ctx := context.Background()

	labels := getDepLabels(dep)
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dep.Name,
			Namespace: ns,
		},

		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
		},
	}
	_, err = c.clientset.CoreV1().Services(ns).Create(ctx, &svc, metav1.CreateOptions{})
	if err != nil {
		fmt.Println("Error creating service for the deployment: ", err.Error())
	}
	// create the corresponding ingress
	return nil
}

func getDepLabels(dep *appsv1.Deployment) map[string]string {
	return dep.Spec.Template.Labels
}

func (c *controller) handleDeploymentAdd(obj interface{}) {
	fmt.Println("Add was called")
	c.queue.Add(obj)
}

func (c *controller) handleDeploymentDelete(obj interface{}) {
	fmt.Println("Delete was called")
	c.queue.Add(obj)
}
