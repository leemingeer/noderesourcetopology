package apihelper

import (
	topologyclientset "github.com/leemingeer/noderesourcetopology/pkg/generated/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	k8sclient "k8s.io/client-go/kubernetes"
)

// APIHelpers represents a set of API helpers for Kubernetes
type APIHelpers interface {
	// GetClient returns a client
	GetClient() (*k8sclient.Clientset, error)

	// GetNode returns the Kubernetes node on which this container is running.
	GetNode(*k8sclient.Clientset, string) (*corev1.Node, error)

	// GetNodes returns all the nodes in the cluster
	GetNodes(*k8sclient.Clientset) (*corev1.NodeList, error)

	// UpdateNode updates the node via the API server using a client.
	UpdateNode(*k8sclient.Clientset, *corev1.Node) error

	// PatchNode updates the node object via the API server using a client.
	PatchNode(*k8sclient.Clientset, string, []JsonPatch) error

	// PatchNodeStatus updates the node status via the API server using a client.
	PatchNodeStatus(*k8sclient.Clientset, string, []JsonPatch) error

	// GetTopologyClient returns a topologyclientset
	GetTopologyClient() (*topologyclientset.Clientset, error)

	// GetPod returns the Kubernetes pod in a namepace with a name.
	GetPod(*k8sclient.Clientset, string, string) (*corev1.Pod, error)
}
