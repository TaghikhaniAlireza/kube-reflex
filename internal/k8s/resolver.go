//internal/k8s/resolver.go
package k8s

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/labels"
)

// PodContext contains enriched metadata about the pod
type PodContext struct {
	PodName    string
	Namespace  string
	NodeName   string
	Image      string
	Labels     map[string]string
	OwnerKind  string // Deployment, DaemonSet, etc.
	OwnerName  string
}

// ResolveContainer finds the pod owning the given container ID
// It iterates through the local cache (Lister), so it's very fast and cheap.
func (k *K8sClient) ResolveContainer(containerID string) (*PodContext, error) {
	// List all pods from the local cache
	// labels.Everything() means "give me all pods"
	pods, err := k.PodLister.List(labels.Everything())
	if err != nil {
		return nil, fmt.Errorf("failed to list pods from cache: %w", err)
	}

	// Iterate over all pods to find the container ID
	// Note: In huge clusters (10k+ pods), we might need a secondary index,
	// but for now, iterating memory is extremely fast (microseconds).
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
		
		// Check Ephemeral Containers (if any)
		for _, container := range pod.Status.EphemeralContainerStatuses {
			if matchContainerID(container.ContainerID, containerID) {
				return extractContext(pod, container.Image), nil
			}
		}
	}

	return nil, fmt.Errorf("pod not found for container ID: %s", containerID)
}

// matchContainerID handles the prefix issue (docker:// vs containerd:// vs short ID)
func matchContainerID(k8sID, searchID string) bool {
	// Falco usually sends the short version (12 chars) or full version.
	// K8s usually stores "containerd://<long_hash>"
	
	// If searchID is empty, it's not a match
	if searchID == "" {
		return false
	}

	// Simple contains check usually works best
	return strings.Contains(k8sID, searchID)
}

// extractContext helps populate the PodContext struct
func extractContext(pod *corev1.Pod, imageName string) *PodContext {
    ctx := &PodContext{
		PodName:   pod.Name,
		Namespace: pod.Namespace,
		NodeName:  pod.Spec.NodeName,
		Image:     imageName,
		Labels:    pod.Labels,
	}
    
    // Attempt to find owner (e.g., ReplicaSet)
    if len(pod.OwnerReferences) > 0 {
        ctx.OwnerKind = pod.OwnerReferences[0].Kind
        ctx.OwnerName = pod.OwnerReferences[0].Name
    }
    
    return ctx
}