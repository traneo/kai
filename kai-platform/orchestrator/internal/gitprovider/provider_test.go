package gitprovider

import (
	"testing"
)

func TestParseRepoOwner(t *testing.T) {
	tests := []struct {
		url       string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{"https://github.com/owner/repo.git", "owner", "repo", false},
		{"https://github.com/owner/repo", "owner", "repo", false},
		{"https://gitlab.com/group/subgroup/project.git", "group/subgroup", "project", false},
		{"https://git.kai.internal/developer/kai-demo.git", "developer", "kai-demo", false},
		{"https://bitbucket.org/user/project.git", "user", "project", false},
		{"", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got, err := ParseRepoOwner(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRepoOwner() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if got.Owner != tt.wantOwner {
				t.Errorf("ParseRepoOwner() owner = %q, want %q", got.Owner, tt.wantOwner)
			}
			if got.Repo != tt.wantRepo {
				t.Errorf("ParseRepoOwner() repo = %q, want %q", got.Repo, tt.wantRepo)
			}
		})
	}
}

func TestInjectToken(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		token    string
		want     string
	}{
		{"with token", "https://github.com/owner/repo.git", "mytoken", "https://git:mytoken@github.com/owner/repo.git"},
		{"no token", "https://github.com/owner/repo.git", "", "https://github.com/owner/repo.git"},
		{"ssh url", "git@github.com:owner/repo.git", "mytoken", "git@github.com:owner/repo.git"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InjectToken(tt.url, tt.token)
			if got != tt.want {
				t.Errorf("InjectToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetectProvider(t *testing.T) {
	r := NewRegistry("")
	r.RegisterBuiltins()

	tests := []struct {
		url  string
		want string
	}{
		{"https://github.com/owner/repo.git", "github"},
		{"https://gitlab.com/group/project.git", "gitlab"},
		{"https://bitbucket.org/user/project.git", "bitbucket"},
		{"https://git.kai.internal/owner/repo.git", ""},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := r.Detect(tt.url)
			if got != tt.want {
				t.Errorf("Detect() = %q, want %q", got, tt.want)
			}
		})
	}
}
