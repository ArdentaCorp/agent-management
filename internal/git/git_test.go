package git

import "testing"

func TestNormalizeURL(t *testing.T) {
	m := NewManager()

	tests := []struct {
		name string
		in   string
		want URLInfo
	}{
		{
			name: "plain repo",
			in:   "https://github.com/user/repo",
			want: URLInfo{
				URL: "https://github.com/user/repo.git",
			},
		},
		{
			name: "tree branch root",
			in:   "https://github.com/user/repo/tree/main",
			want: URLInfo{
				URL:    "https://github.com/user/repo.git",
				Branch: "main",
			},
		},
		{
			name: "tree branch path with query fragment",
			in:   "https://github.com/user/repo/tree/main/skills/tooling?tab=readme#section",
			want: URLInfo{
				URL:    "https://github.com/user/repo.git",
				Branch: "main",
				Path:   "skills/tooling",
			},
		},
		{
			name: "tree encoded slash branch root",
			in:   "https://github.com/user/repo/tree/feature%2Fmy-work",
			want: URLInfo{
				URL:    "https://github.com/user/repo.git",
				Branch: "feature/my-work",
			},
		},
		{
			name: "tree encoded slash branch with path",
			in:   "https://github.com/user/repo/tree/feature%2Fmy-work/skills/tester",
			want: URLInfo{
				URL:    "https://github.com/user/repo.git",
				Branch: "feature/my-work",
				Path:   "skills/tester",
			},
		},
		{
			name: "already git suffix",
			in:   "https://github.com/user/repo.git",
			want: URLInfo{
				URL: "https://github.com/user/repo.git",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.NormalizeURL(tt.in)
			if got != tt.want {
				t.Fatalf("NormalizeURL(%q) = %+v, want %+v", tt.in, got, tt.want)
			}
		})
	}
}
