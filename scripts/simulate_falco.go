// simulate_falco sends dummy Falco alerts to the Kube-Reflex webhook for testing.
//
// Run (with brain listening on :8080):
//   go run scripts/simulate_falco.go
// Or build and run:
//   go build -o simulate_falco scripts/simulate_falco.go && ./simulate_falco
//
// Scenario 1 – FSM (attack chain): Recon → Initial Access → Terminal Shell (same container).
// Scenario 2 – Velocity: 5 rapid Sensitive File Access alerts (same container).
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Payload matches internal/falco.FalcoEventRaw so the webhook decodes it correctly.
type Payload struct {
	Time         time.Time              `json:"time"`
	Rule         string                 `json:"rule"`
	Priority     string                 `json:"priority"`
	Source       string                 `json:"source"`
	Hostname     string                 `json:"hostname"`
	Output       string                 `json:"output"`
	Tags         []string               `json:"tags"`
	OutputFields map[string]interface{} `json:"output_fields"`
}

const (
	defaultURL = "http://localhost:8080/api/v1/alerts"
	container  = "sim-test-container-001"
)

var webhookURL string

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--help" {
		fmt.Fprintf(os.Stderr, "Usage: go run scripts/simulate_falco.go [baseURL]\n")
		fmt.Fprintf(os.Stderr, "  default baseURL: %s\n", defaultURL)
		os.Exit(0)
	}
	webhookURL = defaultURL
	if len(os.Args) > 1 {
		webhookURL = os.Args[1]
	}

	fmt.Println("=== Simulating Falco alerts to", webhookURL, "===")

	// --- 1. FSM: attack chain (Recon → Initial Access → Execution) ---
	fmt.Println("\n--- FSM: Sending attack chain (Recon → Initial Access → Terminal Shell) ---")
	now := time.Now().UTC()
	send(webhookURL, payload(now,
		"Network reconnaissance tool executed in container",
		"Notice",
		[]string{"T1595", "container", "network", "mitre_reconnaissance"},
	))
	time.Sleep(100 * time.Millisecond)
	send(webhookURL, payload(now.Add(2*time.Second),
		"Suspicious download inside container",
		"Warning",
		[]string{"T1568", "container", "file", "mitre_initial_access"},
	))
	time.Sleep(100 * time.Millisecond)
	send(webhookURL, payload(now.Add(4*time.Second),
		"Terminal shell in container",
		"Critical",
		[]string{"T1059", "container", "shell", "mitre_execution"},
	))

	// --- 2. Velocity: 5 rapid Sensitive File Access ---
	fmt.Println("\n--- Velocity: Sending 5 rapid Sensitive File Access alerts ---")
	t0 := time.Now().UTC()
	for i := 0; i < 5; i++ {
		send(webhookURL, payload(t0.Add(time.Duration(i)*100*time.Millisecond),
			"Sensitive file opened for reading by non-trusted program",
			"Warning",
			[]string{"T1555", "container", "filesystem", "mitre_credential_access"},
		))
	}

	fmt.Println("\n=== Done. Check brain logs for FSM chain completion and velocity. ===")
}

func payload(ts time.Time, rule, priority string, tags []string) Payload {
	out := fmt.Sprintf("%s: %s | container_id=%s", ts.Format("15:04:05.000000000"), rule, container)
	return Payload{
		Time:     ts,
		Rule:     rule,
		Priority: priority,
		Source:   "syscall",
		Hostname: "sim-host",
		Output:   out,
		Tags:     tags,
		OutputFields: map[string]interface{}{
			"container.id":                 container,
			"container.name":               "k8s_sim_pod_default_" + container + "_0",
			"container.image.repository":   "busybox",
			"container.image.tag":          "latest",
			"k8s.ns.name":                  "default",
			"k8s.pod.name":                 "sim-pod",
			"evt.type":                     "open",
			"evt.time":                     ts.UnixNano(),
			"proc.name":                    "cat",
			"proc.cmdline":                 "cat /etc/shadow",
			"user.name":                    "root",
			"user.uid":                     "0",
		},
	}
}

func send(url string, p Payload) {
	body, err := json.Marshal(p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal: %v\n", err)
		return
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "POST %s: %v\n", url, err)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusAccepted {
		fmt.Fprintf(os.Stderr, "  %s %s -> %d\n", p.Rule, p.Priority, resp.StatusCode)
		return
	}
	fmt.Printf("  ✓ %s (%s) -> 202 Accepted\n", p.Rule, p.Priority)
}
