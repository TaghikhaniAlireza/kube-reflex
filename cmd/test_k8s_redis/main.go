// cmd/test_k8s_redis/main.go
package main

import (
	"context"
	"log"
	"time"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/k8s"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/parser"
	redisinfra "github.com/TaghikhaniAlireza/kube-reflex/internal/redis"
)

func main() {
	ctx := context.Background()

	// ---------------- K8s Client ----------------
	k8sClient, err := k8s.NewK8sClient()
	if err != nil {
		log.Fatalf("[test] K8s client init failed: %v", err)
	}
	defer k8sClient.Close()
	log.Println("[test] K8s client initialized successfully")

	// ---------------- Redis Client ----------------
	redisClient, err := redisinfra.NewRedisClient()
	if err != nil {
		log.Fatalf("[test] Redis init failed: %v", err)
	}
	redisRepo := redisinfra.NewRepository(redisClient)
	log.Println("[test] Redis client initialized successfully")

	// ---------------- Fetch all container IDs from Redis ----------------
	containerKeys, err := redisClient.Keys(ctx, "container:*").Result()
	if err != nil {
		log.Fatalf("[test] Redis fetch container keys failed: %v", err)
	}

	for _, key := range containerKeys {
		// container:<containerID>
		parts := key[len("container:"):]
		testContainerID := parts
		log.Printf("\n[test] Testing containerID: %s", testContainerID)

		// ---------------- Resolve Pod ----------------
		podCtx, err := k8sClient.Resolve(testContainerID)
		if err != nil {
			log.Printf("[test] K8s Resolve failed: %v", err)
		} else {
			log.Printf("[test] Resolved container %s -> Pod=%s, Namespace=%s, Node=%s",
				testContainerID, podCtx.Pod, podCtx.Namespace, podCtx.Node)
		}

		// ---------------- Update Redis ----------------
		identity := parser.Identity{ContainerID: testContainerID}
		if err := redisRepo.UpdateContainerState(ctx, identity, 50, 15*time.Minute); err != nil {
			log.Printf("[test] Redis update failed: %v", err)
		} else {
			log.Println("[test] Redis updated successfully")
		}

		// ---------------- Check Redis keys ----------------
		keys, err := redisClient.Keys(ctx, "*"+testContainerID+"*").Result()
		if err != nil {
			log.Printf("[test] Redis Keys fetch failed: %v", err)
		} else {
			log.Printf("[test] Redis keys for container %s: %v", testContainerID, keys)
		}
	}

	log.Println("[test] Test finished")
}
