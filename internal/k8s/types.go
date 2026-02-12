//internal/k8s/types.go
package k8s

type Context struct {
	Namespace string
	Pod       string
	Node      string
}

type Resolver interface {
	Resolve(containerID string) (Context, error)
}