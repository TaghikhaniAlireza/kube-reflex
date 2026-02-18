//internal/k8s/client.go
package k8s

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// K8sClient holds the clientset and the pod lister (cache)
type K8sClient struct {
	Clientset *kubernetes.Clientset
	PodLister v1.PodLister
	stopCh    chan struct{}
}

// NewK8sClient initializes the client and starts the informer factory.
// Prioritizes in-cluster config when running inside a pod; falls back to kubeconfig for local dev.
func NewK8sClient() (*K8sClient, error) {
	flag.Parse()

	var config *rest.Config
	var err error

	config, err = rest.InClusterConfig()
	if err == nil {
		log.Println("Using in-cluster Kubernetes config")
	} else {
		var kubeconfig string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
		}
		log.Println("Using kubeconfig for Kubernetes config")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	// Setup Informer Factory (resync every 10 minutes)
	factory := informers.NewSharedInformerFactory(clientset, 10*time.Minute)
	podInformer := factory.Core().V1().Pods()

	// Initialize the client struct
	client := &K8sClient{
		Clientset: clientset,
		PodLister: podInformer.Lister(),
		stopCh:    make(chan struct{}),
	}

	// Start the informer in the background
	log.Println("Starting K8s Informers...")
	factory.Start(client.stopCh)

	// Wait for the cache to sync (CRITICAL step)
	log.Println("Waiting for K8s cache sync...")
	
	// FIX: Handle map return type correctly
	syncResults := factory.WaitForCacheSync(client.stopCh)
	for t, synced := range syncResults {
		if !synced {
			return nil, fmt.Errorf("failed to sync cache for type %v", t)
		}
	}
	
	log.Println("K8s cache synced successfully")

	return client, nil
}

// Close shuts down the informers
func (k *K8sClient) Close() {
	close(k.stopCh)
}