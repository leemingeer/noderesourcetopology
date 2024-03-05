package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DEPRECATED (to be removed in v1beta1): use top level attributes if needed
type TopologyManagerPolicy string

const (
	// Constants of type TopologyManagerPolicy represent policy of the worker
	// node's resource management component. It's TopologyManager in kubelet.
	// DEPRECATED (to be removed in v1beta1): use top level attributes if needed
	// SingleNUMANodeContainerLevel represent single-numa-node policy of
	// the TopologyManager
	SingleNUMANodeContainerLevel TopologyManagerPolicy = "SingleNUMANodeContainerLevel"
	// SingleNUMANodePodLevel enables pod level resource counting, this policy assumes
	// TopologyManager policy single-numa-node also was set on the node
	SingleNUMANodePodLevel TopologyManagerPolicy = "SingleNUMANodePodLevel"
	// Restricted TopologyManager policy was set on the node
	Restricted TopologyManagerPolicy = "Restricted"
	// RestrictedContainerLevel TopologyManager policy was set on the node and TopologyManagerScope was set to pod
	RestrictedContainerLevel TopologyManagerPolicy = "RestrictedContainerLevel"
	// RestrictedPodLevel TopologyManager policy was set on the node and TopologyManagerScope was set to pod
	RestrictedPodLevel TopologyManagerPolicy = "RestrictedPodLevel"
	// BestEffort TopologyManager policy was set on the node
	BestEffort TopologyManagerPolicy = "BestEffort"
	// BestEffort TopologyManager policy was set on the node and TopologyManagerScope was set to container
	BestEffortContainerLevel TopologyManagerPolicy = "BestEffortContainerLevel"
	// BestEffort TopologyManager policy was set on the node and TopologyManagerScope was set to pod
	BestEffortPodLevel TopologyManagerPolicy = "BestEffortPodLevel"
	// None policy is the default policy and does not perform any topology alignment.
	None TopologyManagerPolicy = "None"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope=Cluster,shortName=nrt

type NodeResourceTopology struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// DEPRECATED (to be removed in v1beta1): use top level attributes if needed
	TopologyPolicies []string `json:"topologyPolicies,omitempty"`

	Zones      ZoneList      `json:"zones"`
	Attributes AttributeList `json:"attributes,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type NodeResourceTopologyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []NodeResourceTopology `json:"items"`
}

type ZoneList []Zone
type Zone struct {
	Name       string           `json:"name" protobuf:"bytes,1,opt,name=name"`
	Type       string           `json:"type" protobuf:"bytes,2,opt,name=type"`
	Parent     string           `json:"parent,omitempty" protobuf:"bytes,3,opt,name=parent"`
	Costs      CostList         `json:"costs,omitempty" protobuf:"bytes,4,rep,name=costs"`
	Attributes AttributeList    `json:"attributes,omitempty" protobuf:"bytes,5,rep,name=attributes"`
	Resources  ResourceInfoList `json:"resources,omitempty" protobuf:"bytes,6,rep,name=resources"`
}

type CostList []CostInfo
type CostInfo struct {
	Name  string `json:"name" protobuf:"bytes,1,opt,name=name"`
	Value int64  `json:"value" protobuf:"varint,2,opt,name=value"`
}

type AttributeList []AttributeInfo
type AttributeInfo struct {
	Name  string `json:"name" protobuf:"bytes,1,opt,name=name"`
	Value string `json:"value" protobuf:"bytes,2,opt,name=value"`
}

type ResourceInfoList []ResourceInfo
type ResourceInfo struct {
	// Name of the resource.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Capacity of the resource, corresponding to capacity in node status, i.e.
	// total amount of this resource that the node has.
	Capacity resource.Quantity `json:"capacity" protobuf:"bytes,2,opt,name=capacity"`
	// Allocatable quantity of the resource, corresponding to allocatable in
	// node status, i.e. total amount of this resource available to be used by
	// pods.
	Allocatable resource.Quantity `json:"allocatable" protobuf:"bytes,3,opt,name=allocatable"`
	// Available is the amount of this resource currently available for new (to
	// be scheduled) pods, i.e. Allocatable minus the resources reserved by
	// currently running pods.
	Available resource.Quantity `json:"available" protobuf:"bytes,4,opt,name=available"`
}
