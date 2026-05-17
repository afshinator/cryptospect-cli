package version

import (
	"runtime/debug"
	"strings"
	"testing"
)

func TestFormatVersion(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		settings []debug.BuildSetting
		want     string
	}{
		{
			name:     "no vcs settings",
			base:     "v1.0.0",
			settings: []debug.BuildSetting{},
			want:     "v1.0.0",
		},
		{
			name:     "revision only clean",
			base:     "v1.0.0",
			settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abc1234"}},
			want:     "v1.0.0 (abc1234)",
		},
		{
			name: "revision dirty",
			base: "v1.0.0",
			settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abc1234"},
				{Key: "vcs.modified", Value: "true"},
			},
			want: "v1.0.0 (abc1234-dirty)",
		},
		{
			name: "revision dirty false",
			base: "v1.0.0",
			settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abc1234"},
				{Key: "vcs.modified", Value: "false"},
			},
			want: "v1.0.0 (abc1234)",
		},
		{
			name:     "revision shorter than 7 chars",
			base:     "v1.0.0",
			settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "ab"}},
			want:     "v1.0.0 (ab)",
		},
		{
			name:     "revision exactly 7 chars",
			base:     "v1.0.0",
			settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abc1234"}},
			want:     "v1.0.0 (abc1234)",
		},
		{
			name:     "revision longer than 7 chars truncated",
			base:     "v1.0.0",
			settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abc123456789"}},
			want:     "v1.0.0 (abc1234)",
		},
		{
			name:     "empty revision ignored",
			base:     "v1.0.0",
			settings: []debug.BuildSetting{{Key: "vcs.revision", Value: ""}},
			want:     "v1.0.0",
		},
		{
			name:     "unrelated settings ignored",
			base:     "v1.0.0",
			settings: []debug.BuildSetting{{Key: "GOARCH", Value: "amd64"}},
			want:     "v1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatVersion(tt.base, tt.settings)
			if got != tt.want {
				t.Errorf("formatVersion(%q, ...) = %q, want %q", tt.base, got, tt.want)
			}
		})
	}
}

func TestString(t *testing.T) {
	s := String()
	if s == "" {
		t.Fatal("String() returned empty string")
	}
	if !strings.HasPrefix(s, "v") {
		t.Errorf("String() = %q, want v-prefixed string", s)
	}
}

func TestStringTaggedSkipsCommit(t *testing.T) {
	orig := tagged
	tagged = "true"
	t.Cleanup(func() { tagged = orig })

	origValue := Value
	Value = "v9.9.9"
	t.Cleanup(func() { Value = origValue })

	got := String()
	if got != "v9.9.9" {
		t.Errorf("String() with tagged=true = %q, want %q", got, "v9.9.9")
	}
}
