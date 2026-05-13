package ssh

import (
	"errors"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestDiffInterfaces(t *testing.T) {
	patterns := []string{`^tun`, `^utun`, `^ppp`, `^tap`, `^wg`, `^ipsec`}

	cases := []struct {
		name     string
		current  []string
		baseline []string
		want     []string
	}{
		{
			name:     "no new interfaces",
			current:  []string{"lo0", "en0", "utun0", "utun1"},
			baseline: []string{"lo0", "en0", "utun0", "utun1"},
			want:     nil,
		},
		{
			name:     "ignores non-vpn-like new interfaces (gif, stf)",
			current:  []string{"lo0", "en0", "gif0", "stf0", "utun0"},
			baseline: []string{"lo0", "en0", "utun0"},
			want:     nil,
		},
		{
			name:     "new utun interface flagged",
			current:  []string{"lo0", "en0", "utun0", "utun12"},
			baseline: []string{"lo0", "en0", "utun0"},
			want:     []string{"utun12"},
		},
		{
			name:     "new wireguard and tun both flagged",
			current:  []string{"lo0", "en0", "tun0", "wg0"},
			baseline: []string{"lo0", "en0"},
			want:     []string{"tun0", "wg0"},
		},
		{
			name:     "baseline as trust list suppresses match",
			current:  []string{"lo0", "en0", "tailscale0", "utun9"},
			baseline: []string{"lo0", "en0", "tailscale0", "utun9"},
			want:     nil,
		},
		{
			name:     "interfaces sharing VPN-like prefix still flagged when missing from baseline",
			current:  []string{"lo0", "en0", "wg-corp", "wg-vpn"},
			baseline: []string{"lo0", "en0"},
			want:     []string{"wg-corp", "wg-vpn"},
		},
		{
			name:     "all baselined utuns ignored",
			current:  []string{"lo0", "en0", "utun0", "utun1", "utun2"},
			baseline: []string{"lo0", "en0", "utun0", "utun1", "utun2"},
			want:     nil,
		},
		{
			name:     "empty baseline flags every VPN-like iface",
			current:  []string{"lo0", "en0", "utun0", "tap0"},
			baseline: nil,
			want:     []string{"tap0", "utun0"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DiffInterfaces(tc.current, tc.baseline, patterns)
			want := append([]string{}, tc.want...)
			sort.Strings(want)
			if !reflect.DeepEqual(got, tc.want) && !(len(got) == 0 && len(tc.want) == 0) {
				t.Fatalf("DiffInterfaces(%v, %v, _) = %v, want %v",
					tc.current, tc.baseline, got, tc.want)
			}
		})
	}
}

func TestBaselineRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vpn_baseline.json")

	if _, err := LoadBaseline(path); !errors.Is(err, ErrNoBaseline) {
		t.Fatalf("expected ErrNoBaseline on missing file, got %v", err)
	}

	b, err := CaptureBaseline(path)
	if err != nil {
		t.Fatalf("CaptureBaseline failed: %v", err)
	}
	if len(b.Interfaces) == 0 {
		t.Fatalf("expected at least one interface, got none")
	}
	if b.CapturedAt.IsZero() {
		t.Fatalf("expected non-zero CapturedAt")
	}

	loaded, err := LoadBaseline(path)
	if err != nil {
		t.Fatalf("LoadBaseline failed: %v", err)
	}
	if !reflect.DeepEqual(loaded.Interfaces, b.Interfaces) {
		t.Fatalf("interfaces mismatch after round-trip: got %v, want %v",
			loaded.Interfaces, b.Interfaces)
	}

	if err := ResetBaseline(path); err != nil {
		t.Fatalf("ResetBaseline failed: %v", err)
	}
	if _, err := LoadBaseline(path); !errors.Is(err, ErrNoBaseline) {
		t.Fatalf("expected ErrNoBaseline after reset, got %v", err)
	}

	// Reset on missing file is a no-op (no error).
	if err := ResetBaseline(path); err != nil {
		t.Fatalf("ResetBaseline on missing file should be no-op, got %v", err)
	}
}

func TestMatchAnyIgnoresBadPattern(t *testing.T) {
	if matchAny("utun0", []string{`[`, `^utun`}) != true {
		t.Fatalf("expected match on valid pattern despite bad sibling")
	}
	if matchAny("en0", []string{`[`}) != false {
		t.Fatalf("expected no match when only pattern is invalid")
	}
}
