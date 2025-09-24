package status

import "testing"

func TestStore(t *testing.T) {
	s := NewStore()
	s.EnsureTunnel("db", []string{"5432", "5433"})
	s.Update("db", "5432", "active")
	s.Update("cache", "6379", "connecting")

	snapshot := s.Snapshot()
	if len(snapshot) != 2 {
		t.Fatalf("expected 2 tunnels, got %d", len(snapshot))
	}

	// Ensure copies are independent
	snapshot[0].Ports["5432"] = "mutated"

	snapshot2 := s.Snapshot()
	for _, tun := range snapshot2 {
		if tun.Name == "db" {
			state, ok := tun.Ports["5432"]
			if !ok {
				t.Fatalf("expected db:5432 to exist")
			}
			if state != "active" {
				t.Fatalf("expected db:5432 to be active, got %q", state)
			}
		}
		if tun.Name == "cache" {
			state, ok := tun.Ports["6379"]
			if !ok {
				t.Fatalf("expected cache:6379 to exist")
			}
			if state != "connecting" {
				t.Fatalf("expected cache:6379 to be connecting, got %q", state)
			}
		}
	}
}
