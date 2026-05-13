package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cezmunsta/ssh_ms/log"
	"github.com/cezmunsta/ssh_ms/ssh"
)

var (
	vpnBaselineCmd = &cobra.Command{
		Use:   "vpn-baseline",
		Short: "Manage the VPN interface baseline",
		Long: "Manage the network-interface snapshot used to detect new VPN tunnels " +
			"before connecting. Use `capture` while disconnected from any VPN, `show` " +
			"to inspect the stored snapshot, and `reset` to clear it.",
	}

	vpnBaselineCaptureCmd = &cobra.Command{
		Use:   "capture",
		Short: "Capture the current network interfaces as the safe baseline",
		Long: "Snapshots the currently configured network interfaces and stores them as the baseline. " +
			"Run this while you are NOT connected to any VPN.",
		Run: func(cmd *cobra.Command, args []string) {
			b, err := ssh.CaptureBaseline(cfg.VPNBaselinePath)
			if err != nil {
				log.Fatalf("failed to capture baseline: %v", err)
			}
			fmt.Printf("Baseline captured at %s\n", cfg.VPNBaselinePath)
			fmt.Printf("Interfaces (%d): %v\n", len(b.Interfaces), b.Interfaces)
		},
	}

	vpnBaselineResetCmd = &cobra.Command{
		Use:   "reset",
		Short: "Remove the stored baseline",
		Long:  "Deletes the baseline snapshot. The next `connect` will prompt to capture a new one.",
		Run: func(cmd *cobra.Command, args []string) {
			if err := ssh.ResetBaseline(cfg.VPNBaselinePath); err != nil {
				log.Fatalf("failed to reset baseline: %v", err)
			}
			fmt.Printf("Baseline removed: %s\n", cfg.VPNBaselinePath)
		},
	}

	vpnBaselineShowCmd = &cobra.Command{
		Use:   "show",
		Short: "Display the stored baseline",
		Run: func(cmd *cobra.Command, args []string) {
			b, err := ssh.LoadBaseline(cfg.VPNBaselinePath)
			if errors.Is(err, ssh.ErrNoBaseline) {
				fmt.Printf("No baseline stored at %s\n", cfg.VPNBaselinePath)
				return
			}
			if err != nil {
				log.Fatalf("failed to load baseline: %v", err)
			}
			fmt.Printf("Captured at: %s\n", b.CapturedAt.Format("2006-01-02 15:04:05 MST"))
			fmt.Printf("Hostname:    %s\n", b.Hostname)
			fmt.Printf("Interfaces (%d):\n", len(b.Interfaces))
			for _, name := range b.Interfaces {
				fmt.Printf("  - %s\n", name)
			}
		},
	}
)
