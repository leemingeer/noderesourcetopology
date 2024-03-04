package resourcemonitor

import (
	corev1 "k8s.io/api/core/v1"
	"time"

	topologyv1alpha1 "github.com/leemingeer/noderesourcetopology/pkg/apis/topology/v1alpha1"
)

// Args stores commandline arguments used for resource monitoring
type Args struct {
	PodResourceSocketPath string
	SleepInterval         time.Duration
	Namespace             string
	KubeletConfigURI      string
	APIAuthTokenFile      string
	PodSetFingerprint     bool
}

// ResourceInfo stores information of resources and their corresponding IDs obtained from PodResource API
type ResourceInfo struct {
	Name        corev1.ResourceName
	Data        []string
	NumaNodeIds []int
}

// ContainerResources contains information about the node resources assigned to a container
type ContainerResources struct {
	Name      string
	Resources []ResourceInfo
}

// PodResources contains information about the node resources assigned to a pod
type PodResources struct {
	Name       string
	Namespace  string
	Containers []ContainerResources
}

type ScanResponse struct {
	PodResources []PodResources
	Attributes   topologyv1alpha1.AttributeList
}

// ResourcesScanner gathers all the PodResources from the system, using the podresources API client
type ResourcesScanner interface {
	Scan() (ScanResponse, error)
}

// ResourcesAggregator aggregates resource information based on the received data from underlying hardware and podresource API
type ResourcesAggregator interface {
	Aggregate(podResData []PodResources) topologyv1alpha1.ZoneList
}
