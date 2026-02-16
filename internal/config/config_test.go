package config

import "testing"

func TestSafeNameRoundTrip(t *testing.T) {
	m := &Manager{}
	id := "github:user/repo/path/to/skill"

	safe := m.GetSafeName(id)
	if safe != "github__user__repo__path__to__skill" {
		t.Fatalf("unexpected safe name: %s", safe)
	}

	parsed := m.ParseSafeName(safe)
	if parsed != id {
		t.Fatalf("round-trip mismatch: got %s want %s", parsed, id)
	}
}

func TestGetLinkName(t *testing.T) {
	m := &Manager{}

	tests := []struct {
		id   string
		want string
	}{
		{id: "local:figma-mcp", want: "figma-mcp"},
		{id: "github:user/repo/my-skill", want: "my-skill"},
		{id: "registry:team-skill", want: "team-skill"},
	}

	for _, tt := range tests {
		got := m.GetLinkName(tt.id)
		if got != tt.want {
			t.Fatalf("GetLinkName(%q) = %q, want %q", tt.id, got, tt.want)
		}
	}
}
