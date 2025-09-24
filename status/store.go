package status

import "sync"

// Tunnel represents the status of a single tunnel and its ports.
type Tunnel struct {
	Name  string
	Ports map[string]string
}

// Store keeps track of tunnel status updates for IPC consumers.
type Store struct {
	mu      sync.RWMutex
	tunnels map[string]*Tunnel
}

// NewStore creates an empty status store ready for updates.
func NewStore() *Store {
	return &Store{
		tunnels: make(map[string]*Tunnel),
	}
}

// EnsureTunnel pre-populates entries for a tunnel and its ports.
func (s *Store) EnsureTunnel(name string, ports []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tun, exists := s.tunnels[name]
	if !exists {
		tun = &Tunnel{
			Name:  name,
			Ports: make(map[string]string),
		}
		s.tunnels[name] = tun
	}

	for _, port := range ports {
		if _, ok := tun.Ports[port]; !ok {
			tun.Ports[port] = "pending"
		}
	}
}

// Update records a status change for a tunnel port.
func (s *Store) Update(name string, port string, state string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tun, exists := s.tunnels[name]
	if !exists {
		tun = &Tunnel{
			Name:  name,
			Ports: make(map[string]string),
		}
		s.tunnels[name] = tun
	}
	tun.Ports[port] = state
}

// Snapshot returns a copy of the current tunnel states suitable for external use.
func (s *Store) Snapshot() []Tunnel {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Tunnel, 0, len(s.tunnels))
	for _, tun := range s.tunnels {
		clone := Tunnel{
			Name:  tun.Name,
			Ports: make(map[string]string, len(tun.Ports)),
		}
		for port, state := range tun.Ports {
			clone.Ports[port] = state
		}
		result = append(result, clone)
	}
	return result
}
