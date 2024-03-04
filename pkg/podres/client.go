package podres

import (
	"fmt"
	podresourcesapi "k8s.io/kubelet/pkg/apis/podresources/v1"
	"k8s.io/kubernetes/pkg/kubelet/apis/podresources"
	"log"
	"time"
)

const (
	defaultPodResourcesTimeout = 10 * time.Second
	defaultPodResourcesMaxSize = 1024 * 1024 * 16 // 16 Mb
)

func GetPodResClient(socketPath string) (podresourcesapi.PodResourcesListerClient, error) {
	podResourceClient, _, err := podresources.GetV1Client(socketPath, defaultPodResourcesTimeout, defaultPodResourcesMaxSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create podresource client: %w", err)
	}
	log.Printf("Connected to '%q'!", socketPath)
	return podResourceClient, nil
}
