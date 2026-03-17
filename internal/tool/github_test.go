package tool

import (
	"testing"
)

func TestResolveGHToken_FromGHToken(t *testing.T) {
	t.Setenv("GH_TOKEN", "gh-token-test")
	t.Setenv("GITHUB_TOKEN", "")

	token := ResolveGHToken()
	if token != "gh-token-test" {
		t.Errorf("ResolveGHToken() = %q, want %q", token, "gh-token-test")
	}
}

func TestResolveGHToken_FromGitHubToken(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "github-token-test")

	token := ResolveGHToken()
	if token != "github-token-test" {
		t.Errorf("ResolveGHToken() = %q, want %q", token, "github-token-test")
	}
}

func TestResolveGHToken_PrefersGHToken(t *testing.T) {
	t.Setenv("GH_TOKEN", "preferred")
	t.Setenv("GITHUB_TOKEN", "fallback")

	token := ResolveGHToken()
	if token != "preferred" {
		t.Errorf("ResolveGHToken() = %q, want %q (GH_TOKEN should take priority)", token, "preferred")
	}
}
