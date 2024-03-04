package topologypolicy

import "k8s.io/kubernetes/pkg/kubelet/apis/config"

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

// DetectTopologyPolicy returns string type which present
// both Topology manager policy and scope
func DetectTopologyPolicy(policy string, scope string) TopologyManagerPolicy {
	switch scope {
	case config.PodTopologyManagerScope:
		return detectPolicyPodScope(policy)
	case config.ContainerTopologyManagerScope:
		return detectPolicyContainerScope(policy)
	default:
		return None
	}
}

func detectPolicyPodScope(policy string) TopologyManagerPolicy {
	switch policy {
	case config.SingleNumaNodeTopologyManagerPolicy:
		return SingleNUMANodePodLevel
	case config.RestrictedTopologyManagerPolicy:
		return RestrictedPodLevel
	case config.BestEffortTopologyManagerPolicy:
		return BestEffortPodLevel
	case config.NoneTopologyManagerPolicy:
		return None
	default:
		return None
	}
}

func detectPolicyContainerScope(policy string) TopologyManagerPolicy {
	switch policy {
	case config.SingleNumaNodeTopologyManagerPolicy:
		return SingleNUMANodeContainerLevel
	case config.RestrictedTopologyManagerPolicy:
		return RestrictedContainerLevel
	case config.BestEffortTopologyManagerPolicy:
		return BestEffortContainerLevel
	case config.NoneTopologyManagerPolicy:
		return None
	default:
		return None
	}
}
