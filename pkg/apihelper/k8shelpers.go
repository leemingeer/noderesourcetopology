package apihelper

import (
	"context"
	"encoding/json"

	topologyclientset "github.com/leemingeer/noderesourcetopology/pkg/generated/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// K8sHelpers implements APIHelpers
type K8sHelpers struct {
	Kubeconfig *restclient.Config
}

// GetClient creates and returns a new clientset from given config
func (h K8sHelpers) GetClient() (*k8sclient.Clientset, error) {
	clientset, err := k8sclient.NewForConfig(h.Kubeconfig)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

func (h K8sHelpers) GetTopologyClient() (*topologyclientset.Clientset, error) {
	topologyClient, err := topologyclientset.NewForConfig(h.Kubeconfig)
	if err != nil {
		return nil, err
	}
	return topologyClient, nil
}

// GetNode retrieves one node object.
func (h K8sHelpers) GetNode(cli *k8sclient.Clientset, nodeName string) (*corev1.Node, error) {
	// Get the node object using node name
	node, err := cli.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return node, nil
}

// GetNodes retrieves all the node objects.
func (h K8sHelpers) GetNodes(cli *k8sclient.Clientset) (*corev1.NodeList, error) {
	return cli.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
}

// UpdateNode sends updated node object to the apiserver
func (h K8sHelpers) UpdateNode(c *k8sclient.Clientset, n *corev1.Node) error {
	// Send the updated node to the apiserver.
	_, err := c.CoreV1().Nodes().Update(context.TODO(), n, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (h K8sHelpers) PatchNode(c *k8sclient.Clientset, nodeName string, patches []JsonPatch) error {
	if len(patches) > 0 {
		data, err := json.Marshal(patches)
		if err == nil {
			_, err = c.CoreV1().Nodes().Patch(context.TODO(), nodeName, types.JSONPatchType, data, metav1.PatchOptions{})
		}
		return err
	}
	return nil
}

func (h K8sHelpers) PatchNodeStatus(c *k8sclient.Clientset, nodeName string, patches []JsonPatch) error {
	if len(patches) > 0 {
		data, err := json.Marshal(patches)
		if err == nil {
			_, err = c.CoreV1().Nodes().Patch(context.TODO(), nodeName, types.JSONPatchType, data, metav1.PatchOptions{}, "status")
		}
		return err
	}
	return nil

}

func (h K8sHelpers) GetPod(cli *k8sclient.Clientset, namespace string, podName string) (*corev1.Pod, error) {
	// Get the node object using pod name
	pod, err := cli.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return pod, nil
}

// GetKubeconfig returns the kubeconfig for the cluster
func GetKubeconfig(path string) (*restclient.Config, error) {
	if path == "" {
		return restclient.InClusterConfig()
	}
	return clientcmd.BuildConfigFromFlags("", path)
}
