package model

import (
	"testing"
)

func TestChannel_IsForwardKeyMode(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{"empty key", "", false},
		{"single sentinel exact", "$FORWARD_KEY", true},
		{"single sentinel with surrounding whitespace", "  $FORWARD_KEY  ", true},
		{"single real key", "sk-realkey", false},
		{"sentinel as substring (suffix)", "$FORWARD_KEY_extra", false},
		{"sentinel as substring (prefix)", "extra$FORWARD_KEY", false},
		{"multi-key with sentinel first", "$FORWARD_KEY\nrealkey2\nrealkey3", true},
		{"multi-key with sentinel middle", "realkey1\n$FORWARD_KEY\nrealkey3", true},
		{"multi-key with sentinel last", "realkey1\nrealkey2\n$FORWARD_KEY", true},
		{"multi-key all real", "realkey1\nrealkey2\nrealkey3", false},
		{"multi-key with sentinel and whitespace", "realkey1\n  $FORWARD_KEY  \nrealkey3", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &Channel{Key: tc.key}
			if got := c.IsForwardKeyMode(); got != tc.want {
				t.Fatalf("IsForwardKeyMode()=%v, want %v (key=%q)", got, tc.want, tc.key)
			}
		})
	}

	t.Run("nil channel", func(t *testing.T) {
		var c *Channel
		if c.IsForwardKeyMode() {
			t.Fatalf("nil channel should not be BYOK")
		}
	})
}
