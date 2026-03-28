package widgets

import "testing"

func TestResolveActiveSectionFallsBackToFirstAvailable(t *testing.T) {
	t.Parallel()

	got := ResolveActiveSection("Warnings", []string{"Summary", "Security"}, "Summary")
	if got != "Summary" {
		t.Fatalf("expected first available section, got %q", got)
	}
}

func TestMoveActiveSectionKeepsCurrentWhenDirectionLeavesBounds(t *testing.T) {
	t.Parallel()

	got := MoveActiveSection("Security", []string{"Summary", "Security"}, 1, "Summary")
	if got != "Security" {
		t.Fatalf("expected move to stay on current section at boundary, got %q", got)
	}
}

func TestBoundaryActiveSectionReturnsEmptyFallbackWhenUnavailable(t *testing.T) {
	t.Parallel()

	got := BoundaryActiveSection(nil, true, "")
	if got != "" {
		t.Fatalf("expected empty fallback, got %q", got)
	}
}
