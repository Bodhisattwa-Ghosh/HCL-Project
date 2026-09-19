package service

import (
	"testing"
	"time"

	"github.com/Bodhisattwa-Ghosh/HCL-Project/backend/internal/domain"
)

func TestIsClosed(t *testing.T) {
	now := time.Date(2026, time.September, 19, 12, 0, 0, 0, time.UTC)
	passed := now.Add(-time.Minute)
	future := now.Add(time.Minute)
	if !isClosed(domain.Poll{ClosesAt: &passed}, now) {
		t.Fatal("poll past its close time should be closed")
	}
	if isClosed(domain.Poll{ClosesAt: &future}, now) {
		t.Fatal("poll before its close time should be open")
	}
	closedAt := now
	if !isClosed(domain.Poll{ClosedAt: &closedAt}, now) {
		t.Fatal("explicitly closed poll should be closed")
	}
}
