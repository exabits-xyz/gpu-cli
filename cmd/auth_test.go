package cmd

import (
	"testing"

	"github.com/exabits-xyz/gpu-cli/internal/api"
	"github.com/spf13/viper"
)

func TestBuildAuthURLFromDefaultAPIBase(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	got := buildAuthURL(api.DefaultBaseURL(), "state-code")
	want := "https://gpu.exascalelabs.ai/login?state=state-code"
	if got != want {
		t.Fatalf("buildAuthURL = %q, want %q", got, want)
	}
}

func TestBuildAuthURLUsesOverride(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("auth_url", "https://example.test/login?source=cli")

	got := buildAuthURL(api.DefaultBaseURL(), "state code")
	want := "https://example.test/login?source=cli&state=state+code"
	if got != want {
		t.Fatalf("buildAuthURL = %q, want %q", got, want)
	}
}
