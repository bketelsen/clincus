package tool

import (
	"os"
	"os/exec"
	"strings"
)

// ResolveGHToken resolves a GitHub token from the host environment.
// Priority: GH_TOKEN env > GITHUB_TOKEN env > `gh auth token` CLI output.
// Returns "" if no token can be found.
func ResolveGHToken() string {
	if token := os.Getenv("GH_TOKEN"); token != "" {
		return token
	}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token
	}

	out, err := exec.Command("gh", "auth", "token").Output()
	if err == nil {
		token := strings.TrimSpace(string(out))
		if token != "" {
			return token
		}
	}

	return ""
}
