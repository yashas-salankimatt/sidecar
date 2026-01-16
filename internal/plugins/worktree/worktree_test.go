package worktree

import (
	"testing"
)

func TestParseWorktreeList(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		mainWorkdir string
		wantCount   int
		wantNames   []string
		wantBranch  []string
	}{
		{
			name: "single worktree",
			output: `worktree /home/user/project
HEAD abc123
branch refs/heads/main

worktree /home/user/project-feature
HEAD def456
branch refs/heads/feature
`,
			mainWorkdir: "/home/user/project",
			wantCount:   1,
			wantNames:   []string{"project-feature"},
			wantBranch:  []string{"feature"},
		},
		{
			name: "multiple worktrees",
			output: `worktree /home/user/project
HEAD abc123
branch refs/heads/main

worktree /home/user/feature-a
HEAD def456
branch refs/heads/feature-a

worktree /home/user/feature-b
HEAD ghi789
branch refs/heads/feature-b
`,
			mainWorkdir: "/home/user/project",
			wantCount:   2,
			wantNames:   []string{"feature-a", "feature-b"},
			wantBranch:  []string{"feature-a", "feature-b"},
		},
		{
			name: "detached head",
			output: `worktree /home/user/project
HEAD abc123
branch refs/heads/main

worktree /home/user/detached
HEAD def456
detached
`,
			mainWorkdir: "/home/user/project",
			wantCount:   1,
			wantNames:   []string{"detached"},
			wantBranch:  []string{"(detached)"},
		},
		{
			name:        "empty output",
			output:      "",
			mainWorkdir: "/home/user/project",
			wantCount:   0,
			wantNames:   nil,
			wantBranch:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worktrees, err := parseWorktreeList(tt.output, tt.mainWorkdir)
			if err != nil {
				t.Fatalf("parseWorktreeList() error = %v", err)
			}

			if len(worktrees) != tt.wantCount {
				t.Errorf("got %d worktrees, want %d", len(worktrees), tt.wantCount)
			}

			for i, wt := range worktrees {
				if i < len(tt.wantNames) && wt.Name != tt.wantNames[i] {
					t.Errorf("worktree[%d].Name = %q, want %q", i, wt.Name, tt.wantNames[i])
				}
				if i < len(tt.wantBranch) && wt.Branch != tt.wantBranch[i] {
					t.Errorf("worktree[%d].Branch = %q, want %q", i, wt.Branch, tt.wantBranch[i])
				}
			}
		})
	}
}

func TestValidateBranchName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValid bool
		wantError string // Substring to look for in errors
	}{
		// Valid names
		{name: "simple", input: "feature-auth", wantValid: true},
		{name: "with-slash", input: "fix/bug-123", wantValid: true},
		{name: "with-dot", input: "release-1.0", wantValid: true},
		{name: "nested-slash", input: "user/feature/auth", wantValid: true},

		// Invalid names
		{name: "empty", input: "", wantValid: false, wantError: "cannot be empty"},
		{name: "starts-with-dot", input: ".hidden", wantValid: false, wantError: "start with '.'"},
		{name: "starts-with-dash", input: "-dash", wantValid: false, wantError: "start with '-'"},
		{name: "ends-with-slash", input: "foo/", wantValid: false, wantError: "end with '/'"},
		{name: "ends-with-lock", input: "foo.lock", wantValid: false, wantError: "end with '.lock'"},
		{name: "double-dot", input: "foo..bar", wantValid: false, wantError: "contain '..'"},
		{name: "double-slash", input: "foo//bar", wantValid: false, wantError: "contain '//'"},
		{name: "slash-dot", input: "foo/.bar", wantValid: false, wantError: "contain '/.'"},
		{name: "single-at", input: "@", wantValid: false, wantError: "exactly '@'"},
		{name: "at-brace", input: "foo@{bar}", wantValid: false, wantError: "contain '@{'"},
		{name: "with-space", input: "has space", wantValid: false, wantError: "space"},
		{name: "with-tilde", input: "foo~bar", wantValid: false, wantError: "~"},
		{name: "with-colon", input: "foo:bar", wantValid: false, wantError: ":"},
		{name: "with-question", input: "foo?bar", wantValid: false, wantError: "?"},
		{name: "with-asterisk", input: "foo*bar", wantValid: false, wantError: "*"},
		{name: "with-bracket", input: "foo[bar", wantValid: false, wantError: "["},
		{name: "with-backslash", input: "foo\\bar", wantValid: false, wantError: "\\"},
		{name: "with-caret", input: "foo^bar", wantValid: false, wantError: "^"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, errors := ValidateBranchName(tt.input)
			if valid != tt.wantValid {
				t.Errorf("ValidateBranchName(%q) valid = %v, want %v, errors: %v", tt.input, valid, tt.wantValid, errors)
			}
			if !tt.wantValid && tt.wantError != "" {
				found := false
				for _, e := range errors {
					if contains(e, tt.wantError) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ValidateBranchName(%q) expected error containing %q, got %v", tt.input, tt.wantError, errors)
				}
			}
		})
	}
}

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "already-valid", input: "feature-auth", want: "feature-auth"},
		{name: "space-to-dash", input: "hello world", want: "hello-world"},
		{name: "underscore-to-dash", input: "hello_world", want: "hello-world"},
		{name: "remove-lock-suffix", input: "foo.lock", want: "foo"},
		{name: "double-lock-suffix", input: "foo.lock.lock", want: "foo"},
		{name: "collapse-dots", input: "foo..bar", want: "foo.bar"},
		{name: "collapse-slashes", input: "foo//bar", want: "foo/bar"},
		{name: "remove-slash-dot", input: "foo/.bar", want: "foo/bar"},
		{name: "remove-at-brace", input: "foo@{bar}", want: "foobar}"},
		{name: "single-at", input: "@", want: "at"},
		{name: "leading-dot", input: ".hidden", want: "hidden"},
		{name: "leading-dash", input: "-dash", want: "dash"},
		{name: "trailing-slash", input: "foo/", want: "foo"},
		{name: "collapse-dashes", input: "foo--bar", want: "foo-bar"},
		{name: "complex", input: "hello world..foo/.bar//baz.lock", want: "hello-world.foo/bar/baz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeBranchName(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeBranchName(%q) = %q, want %q", tt.input, got, tt.want)
			}
			// Verify output is valid (unless empty)
			if got != "" {
				valid, errors := ValidateBranchName(got)
				if !valid {
					t.Errorf("SanitizeBranchName(%q) produced invalid output %q: %v", tt.input, got, errors)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
