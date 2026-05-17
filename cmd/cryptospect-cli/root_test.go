package main

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

var _ = cobra.Command{}

func TestDetailFlagValidation(t *testing.T) {
	tests := []struct {
		detail  string
		wantErr bool
	}{
		{"basic", false},
		{"extended", false},
		{"full", false},
		{"invalid", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.detail, func(t *testing.T) {
			cmd := NewRootCommand()
			// Set args and execute
			cmd.SetArgs([]string{"--detail", tt.detail})
			err := cmd.Execute()
			if tt.wantErr && err == nil {
				t.Errorf("expected error for detail=%q, got nil", tt.detail)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for detail=%q: %v", tt.detail, err)
			}
		})
	}
}

func TestNewRootCommand(t *testing.T) {
	cmd := NewRootCommand()
	if cmd == nil {
		t.Fatal("NewRootCommand returned nil")
	}
	if cmd.Use != "cryptospect-cli" {
		t.Errorf("expected Use to be 'cryptospect-cli', got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description to be non-empty")
	}
	if cmd.Long == "" {
		t.Error("expected Long description to be non-empty")
	}
}

func TestRootCommandFlags(t *testing.T) {
	cmd := NewRootCommand()

	// Check --verbose flag
	verboseFlag := cmd.PersistentFlags().Lookup("verbose")
	if verboseFlag == nil {
		t.Fatal("--verbose flag not defined")
	}
	if verboseFlag.Shorthand != "v" {
		t.Errorf("expected shorthand 'v' for --verbose, got %q", verboseFlag.Shorthand)
	}
	if verboseFlag.DefValue != "false" {
		t.Errorf("expected default value 'false' for --verbose, got %q", verboseFlag.DefValue)
	}

	// Check --detail flag
	detailFlag := cmd.PersistentFlags().Lookup("detail")
	if detailFlag == nil {
		t.Fatal("--detail flag not defined")
	}
	if detailFlag.DefValue != "basic" {
		t.Errorf("expected default value 'basic' for --detail, got %q", detailFlag.DefValue)
	}

	// Check --output flag
	outputFlag := cmd.PersistentFlags().Lookup("output")
	if outputFlag == nil {
		t.Fatal("--output flag not defined")
	}
	if outputFlag.Shorthand != "o" {
		t.Errorf("expected shorthand 'o' for --output, got %q", outputFlag.Shorthand)
	}
	if outputFlag.DefValue != "json" {
		t.Errorf("expected default value 'json' for --output, got %q", outputFlag.DefValue)
	}

	// Check --api-key flag
	apiKeyFlag := cmd.PersistentFlags().Lookup("api-key")
	if apiKeyFlag == nil {
		t.Fatal("--api-key flag not defined")
	}
	if apiKeyFlag.DefValue != "" {
		t.Errorf("expected empty default value for --api-key, got %q", apiKeyFlag.DefValue)
	}
}

func TestListMetricsSubcommandExists(t *testing.T) {
	cmd := NewRootCommand()
	found := false
	for _, sub := range cmd.Commands() {
		t.Logf("subcommand: %q", sub.Use)
		if sub.Use == "list-metrics" {
			found = true
			break
		}
	}
	if !found {
		t.Error("list-metrics subcommand not found")
	}
}

func TestRootVersion(t *testing.T) {
	cmd := NewRootCommand()
	if cmd.Version == "" {
		t.Fatal("Version is empty")
	}
	if !strings.HasPrefix(cmd.Version, "v") {
		t.Errorf("Version = %q, want v-prefixed string", cmd.Version)
	}
}

func TestCommandGroups(t *testing.T) {
	cmd := NewRootCommand()

	groups := map[string]bool{}
	for _, g := range cmd.Groups() {
		groups[g.ID] = true
	}
	for _, want := range []string{"metrics", "utility"} {
		if !groups[want] {
			t.Errorf("group %q not found on root command", want)
		}
	}

	metricCmds := map[string]bool{
		"market-regime": true, "liquidity-pulse": true, "flow-tension": true,
		"stablecoin-power": true, "market-breadth": true, "momentum-divergence": true,
	}
	utilityCmds := map[string]bool{"list-metrics": true, "cache-clear": true}

	for _, sub := range cmd.Commands() {
		if metricCmds[sub.Use] {
			if sub.GroupID != "metrics" {
				t.Errorf("%s GroupID = %q, want %q", sub.Use, sub.GroupID, "metrics")
			}
		}
		if utilityCmds[sub.Use] {
			if sub.GroupID != "utility" {
				t.Errorf("%s GroupID = %q, want %q", sub.Use, sub.GroupID, "utility")
			}
		}
	}
}

func TestCompletionHidden(t *testing.T) {
	cmd := NewRootCommand()
	if !cmd.CompletionOptions.HiddenDefaultCmd {
		t.Error("CompletionOptions.HiddenDefaultCmd should be true")
	}
}
