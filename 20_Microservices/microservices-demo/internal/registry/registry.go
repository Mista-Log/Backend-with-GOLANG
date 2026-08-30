// Package registry implements the minimal service-registry idea from the
// guide's Service Discovery section — good enough to demonstrate the
// pattern; a real deployment would use Consul, etcd, or Kubernetes' own
// service discovery instead of hand-rolling this.
package registry

import (
	"fmt"
	"math/rand"
	"sync"
)

type Registry struct {
	mu        sync.RWMutex
	instances map[string][]string
}

func New() *Registry {
	return &Registry{instances: make(map[string][]string)}
}

func (r *Registry) Register(service, address string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.instances[service] {
		if existing == address {
			return // already registered — avoid duplicate entries on retry
		}
	}
	r.instances[service] = append(r.instances[service], address)
}

// Discover picks one instance at random — the simplest possible load
// balancing strategy, sufficient for this demo (real systems use
// round-robin, least-connections, or latency-aware strategies instead).
func (r *Registry) Discover(service string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	addrs := r.instances[service]
	if len(addrs) == 0 {
		return "", fmt.Errorf("no instances of %q registered", service)
	}
	return addrs[rand.Intn(len(addrs))], nil
}
