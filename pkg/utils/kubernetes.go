package utils

import "os"

var nodeName string

// NodeName returns the name of the k8s node we're running on.
func NodeName() string {
	if nodeName == "" {
		nodeName = os.Getenv("NODE_NAME")
	}
	return nodeName
}
