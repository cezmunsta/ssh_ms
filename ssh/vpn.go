package ssh

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/cezmunsta/ssh_ms/log"
)

// Baseline is the persisted snapshot of network-interface names captured
// while the user has confirmed they are disconnected from any VPN. New
// interfaces appearing after this baseline (and matching the VPN pattern
// list) trigger a confirmation prompt before connect.
type Baseline struct {
	CapturedAt time.Time `json:"captured_at"`
	Hostname   string    `json:"hostname"`
	Interfaces []string  `json:"interfaces"`
}

// ErrNoBaseline is returned by LoadBaseline when no snapshot exists.
var ErrNoBaseline = errors.New("no baseline snapshot found")

// currentInterfaces returns the names of all currently configured
// network interfaces, sorted for deterministic comparisons.
func currentInterfaces() ([]string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to list interfaces: %w", err)
	}
	names := make([]string, 0, len(ifaces))
	for _, i := range ifaces {
		names = append(names, i.Name)
	}
	sort.Strings(names)
	return names, nil
}

// LoadBaseline reads the baseline snapshot from path. Returns
// ErrNoBaseline when the file does not exist.
func LoadBaseline(path string) (*Baseline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoBaseline
		}
		return nil, fmt.Errorf("failed to read baseline %s: %w", path, err)
	}
	var b Baseline
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("failed to parse baseline %s: %w", path, err)
	}
	return &b, nil
}

// SaveBaseline persists the snapshot to path with 0600 permissions.
func SaveBaseline(path string, b *Baseline) error {
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal baseline: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write baseline %s: %w", path, err)
	}
	return nil
}

// ResetBaseline removes the baseline file. Returns nil if the file does
// not exist.
func ResetBaseline(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// CaptureBaseline snapshots the current network interfaces and writes
// the result to path. The returned Baseline is the value just written.
func CaptureBaseline(path string) (*Baseline, error) {
	names, err := currentInterfaces()
	if err != nil {
		return nil, err
	}
	host, _ := os.Hostname()
	b := &Baseline{
		CapturedAt: time.Now().UTC(),
		Hostname:   host,
		Interfaces: names,
	}
	if err := SaveBaseline(path, b); err != nil {
		return nil, err
	}
	return b, nil
}

// matchAny returns true if name matches any of the regex patterns.
// Patterns that fail to compile are skipped with a warning.
func matchAny(name string, patterns []string) bool {
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			log.Warningf("invalid regex pattern %q: %v", p, err)
			continue
		}
		if re.MatchString(name) {
			return true
		}
	}
	return false
}

// DiffInterfaces returns the names of interfaces in current that are not
// present in baseline and match at least one of vpnPatterns. The
// baseline doubles as the trust list — to permanently whitelist an
// interface, capture a fresh baseline that includes it. The result is
// sorted.
func DiffInterfaces(current, baseline, vpnPatterns []string) []string {
	known := make(map[string]struct{}, len(baseline))
	for _, n := range baseline {
		known[n] = struct{}{}
	}

	var out []string
	for _, name := range current {
		if _, seen := known[name]; seen {
			continue
		}
		if !matchAny(name, vpnPatterns) {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// promptYesNo asks the user for confirmation on stdin. Returns true on
// y/yes/1, false on n/no/0. EOF or unrecognised input returns false.
func promptYesNo(question string) bool {
	fmt.Printf("%s (y/n) ", question)
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			fmt.Println()
		} else {
			fmt.Fprintf(os.Stderr, "error reading input: %v\n", err)
		}
		return false
	}
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes", "1":
		return true
	default:
		return false
	}
}

// EnforceBaseline runs the baseline-aware VPN check. If no snapshot
// exists, it prompts the user to confirm they are VPN-disconnected and
// captures one on a positive answer. If a snapshot exists, it computes
// the delta of VPN-like interfaces and prompts before continuing.
//
// Returns true when it is safe to continue with the connection. False
// indicates the user declined or input could not be read.
func EnforceBaseline(baselinePath string, vpnPatterns []string) bool {
	current, err := currentInterfaces()
	if err != nil {
		log.Warningf("VPN baseline check skipped: %v", err)
		return true
	}

	baseline, err := LoadBaseline(baselinePath)
	if errors.Is(err, ErrNoBaseline) {
		fmt.Println()
		fmt.Println("==================== VPN baseline setup ====================")
		fmt.Println("No baseline of your network interfaces is on file yet.")
		fmt.Println()
		fmt.Println("ssh_ms uses this baseline to detect new VPN tunnels (tun/utun/")
		fmt.Println("ppp/tap/wg/ipsec) that appear after the snapshot. The goal is to")
		fmt.Println("prevent connecting to a sensitive customer host while another")
		fmt.Println("customer's VPN is active — that has caused security incidents in")
		fmt.Println("the past because traffic can be tunneled through the wrong network.")
		fmt.Println()
		fmt.Println("Before capturing the baseline, please make sure that ALL customer")
		fmt.Println("VPNs are disconnected. Trusted always-on interfaces (corporate")
		fmt.Println("VPN, Tailscale, etc.) will be included in the snapshot and never")
		fmt.Println("flagged again. To trust a new VPN later, simply re-capture the")
		fmt.Println("baseline while that VPN is connected and nothing else is.")
		fmt.Println()
		fmt.Println("Baseline commands:")
		fmt.Println("  ssh_ms vpn-baseline capture   # snapshot current interfaces")
		fmt.Println("  ssh_ms vpn-baseline reset     # forget the baseline")
		fmt.Println("  ssh_ms vpn-baseline show      # display the stored snapshot")
		fmt.Println("============================================================")
		fmt.Println()
		if !promptYesNo("Are you currently disconnected from every customer VPN?") {
			fmt.Println()
			fmt.Println("Connection aborted. Disconnect from your VPN first, then either")
			fmt.Println("re-run this command or capture the baseline manually with:")
			fmt.Println("  ssh_ms vpn-baseline capture")
			return false
		}
		b, err := CaptureBaseline(baselinePath)
		if err != nil {
			log.Warningf("failed to capture baseline: %v", err)
			return false
		}
		fmt.Println()
		fmt.Printf("Baseline captured (%d interfaces) at:\n  %s\n", len(b.Interfaces), baselinePath)
		fmt.Println("Future connects will warn if new VPN-like interfaces appear.")
		fmt.Println()
		return true
	}
	if err != nil {
		log.Warningf("baseline check failed: %v", err)
		return true
	}

	delta := DiffInterfaces(current, baseline.Interfaces, vpnPatterns)
	if len(delta) == 0 {
		return true
	}

	fmt.Println()
	fmt.Println("==================== VPN safety check ======================")
	fmt.Println("New VPN-like network interfaces detected since baseline:")
	for _, name := range delta {
		fmt.Printf("  - %s\n", name)
	}
	fmt.Println()
	fmt.Println("This usually means a VPN client connected after your baseline was")
	fmt.Println("captured. If the VPN you're on belongs to a DIFFERENT customer than")
	fmt.Println("the host you're about to reach, your SSH traffic can be tunneled")
	fmt.Println("through the wrong customer's network — this is a known cause of")
	fmt.Println("cross-customer routing incidents.")
	fmt.Println()
	fmt.Println("Proceed only if you've verified that the active VPN matches the")
	fmt.Println("customer this host belongs to.")
	fmt.Println()
	fmt.Println("If this VPN is something you want ssh_ms to trust from now on")
	fmt.Println("(e.g. a corporate always-on VPN you just installed), connect ONLY")
	fmt.Println("to that VPN and re-capture the baseline:")
	fmt.Println("  ssh_ms vpn-baseline capture")
	fmt.Println()
	fmt.Println("To skip this check for one run, re-run with --check-vpn=false.")
	fmt.Println("============================================================")
	fmt.Println()
	return promptYesNo("Proceed with the connection?")
}
