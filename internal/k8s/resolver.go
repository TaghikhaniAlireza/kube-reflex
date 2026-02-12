//internal/k8s/resolver.go
package k8s

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/labels"
)

// Resolve finds the pod owning the given container ID and returns a lightweight context.
// It implements the k8s.Resolver interface defined in types.go.
func (k *K8sClient) Resolve(containerID string) (Context, error) {
	// List all pods from the local cache
	pods, err := k.PodLister.List(labels.Everything())
	if err != nil {
		return Context{}, fmt.Errorf("failed to list pods from cache: %w", err)
	}

	for _, pod := range pods {
		// Check normal containers
		for _, container := range pod.Status.ContainerStatuses {
			if matchContainerID(container.ContainerID, containerID) {
				return Context{
					Namespace: pod.Namespace,
					Pod:       pod.Name,
					Node:      pod.Spec.NodeName,
				}, nil
			}
		}
	}

	return Context{}, fmt.Errorf("pod not found for container ID: %s", containerID)
}

func matchContainerID(k8sID, searchID string) bool {
	if searchID == "" {
		return false
	}
	// Handles "containerd://<hash>" vs "<short_hash>"
	return strings.Contains(k8sID, searchID)
}