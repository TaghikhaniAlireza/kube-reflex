//internal/k8s/resolver.go
package k8s

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1" // FIX: Added this import alias
	"k8s.io/apimachinery/pkg/labels"
)

// PodContext contains enriched metadata about the pod
type PodContext struct {
	PodName    string
	Namespace  string
	NodeName   string
	Image      string
	Labels     map[string]string
	OwnerKind  string
	OwnerName  string
}

// ResolveContainer finds the pod owning the given container ID
func (k *K8sClient) ResolveContainer(containerID string) (*PodContext, error) {
	// List all pods from the local cache
	pods, err := k.PodLister.List(labels.Everything())
	if err != nil {
		return nil, fmt.Errorf("failed to list pods from cache: %w", err)
	}

	for _, pod := range pods {
		// Check InitContainers
		for _, container := range pod.Status.InitContainerStatuses {
			if matchContainerID(container.ContainerID, containerID) {
				return extractContext(pod, container.Image), nil
			}
		}

		// Check Normal Containers
		for _, container := range pod.Status.ContainerStatuses {
			if matchContainerID(container.ContainerID, containerID) {
				return extractContext(pod, container.Image), nil
			}
		}
		
		// Check Ephemeral Containers
		for _, container := range pod.Status.EphemeralContainerStatuses {
			if matchContainerID(container.ContainerID, containerID) {
				return extractContext(pod, container.Image), nil
			}
		}
	}

	return nil, fmt.Errorf("pod not found for container ID: %s", containerID)
}

func matchContainerID(k8sID, searchID string) bool {
	if searchID == "" {
		return false
	}
	// Handles "containerd://<hash>" vs "<short_hash>"
	return strings.Contains(k8sID, searchID)
}

func extractContext(pod *corev1.Pod, imageName string) *PodContext {
    ctx := &PodContext{
		PodName:   pod.Name,
		Namespace: pod.Namespace,
		NodeName:  pod.Spec.NodeName,
		Image:     imageName,
		Labels:    pod.Labels,
	}
    
    if len(pod.OwnerReferences) > 0 {
        ctx.OwnerKind = pod.OwnerReferences[0].Kind
        ctx.OwnerName = pod.OwnerReferences[0].Name
    }
    
 return ctx
}