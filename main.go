package main

import (
	"context"
	"fmt"
	clientset "github.com/leemingeer/noderesourcetopology/pkg/generated/clientset/versioned"
	"github.com/leemingeer/noderesourcetopology/pkg/generated/informers/externalversions"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"time"
)

func main() {
	// step1, need config obj
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		log.Fatalln(err)
	}
	// then need client
	clientset, err := clientset.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
	}

	list, err := clientset.TopologyV1alpha1().NodeResourceTopologies().List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Fatalln(err)
	}
	for _, t := range list.Items {
		println("clientSet:", t.Name)
	}

	factory := externalversions.NewSharedInformerFactory(clientset, 5*time.Second)
	nrtInformer := factory.Topology().V1alpha1().NodeResourceTopologies()

	informer := nrtInformer.Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			println("AddFunc", "obj", obj)
		},
		DeleteFunc: func(obj interface{}) {
			println("DeleteFunc", "obj", obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			println("UpdateFunc", "oldObj", oldObj, "newObj", newObj)
		},
	})

	stopCh := make(chan struct{})
	factory.Start(stopCh)
	// sync to local cache
	factory.WaitForCacheSync(stopCh)

	ret, err := nrtInformer.Lister().List(labels.Everything())
	if err != nil {
		log.Fatalln(err)
	}
	for _, t := range ret {
		fmt.Printf("nrt informer lister: %v\n", t.GetName())
	}
	<-stopCh
}
