package main

import (
	"fmt"
	"os"

	"github.com/alexmaze/clink/config"
	"github.com/alexmaze/clink/lib/color"
	"github.com/alexmaze/clink/lib/icon"
	"github.com/alexmaze/clink/lib/spinner"
	"github.com/alexmaze/clink/lib/sshutil"
)

// CheckStatus represents the health status of a single deployed item.
type CheckStatus int

const (
	CheckOK      CheckStatus = iota // destination is correctly established
	CheckWrong                      // destination exists but is wrong (e.g. symlink points elsewhere)
	CheckMissing                    // destination does not exist
	CheckError                      // an unexpected error occurred during the check
)

// itemCheckResult holds the result of checking one RuleItem.
type itemCheckResult struct {
	item   *config.RuleItem
	status CheckStatus
	detail string // human-readable description shown on the right side
}

// runCheck implements the --check command: loads config (read-only, no backup
// prompt), checks each item, and prints a colour-coded summary.
func runCheck(opts ClinkOpts) {
	cfg, err := config.ReadConfigForCheck(opts.ConfigPath, opts.Rules)
	if err != nil {
		sp := spinner.New()
		sp.Failedf("%v", err)
		os.Exit(1)
	}

	sp := spinner.New()

	totalRules := len(cfg.Rules)
	var totalOK, totalWrong, totalMissing, totalErrors int

	for i, rule := range cfg.Rules {
		modeLabel := config.BuildModeLabel(cfg.ConfigFile, rule)
		sp.CheckPoint(icon.IconInfo, color.ColorCyan,
			fmt.Sprintf("[%d/%d] %s  [%s]", i+1, totalRules, rule.Name, modeLabel),
			color.ColorReset)

		// For SSH rules, establish a connection once per rule.
		var sshClient *sshutil.Client
		if rule.Mode == config.ModeSSH {
			srv := cfg.SSHServers[rule.SSH]
			sp.Start(fmt.Sprintf("  Connecting to %s@%s ...", srv.User, srv.Host))
			sshClient, err = sshutil.NewClient(srv)
			sp.Stop()
			if err != nil {
				sp.CheckPoint(icon.IconCross, color.ColorRed,
					fmt.Sprintf("  SSH connect failed: %s", err.Error()),
					color.ColorReset)
				// Count all items in this rule as errors.
				for range rule.Items {
					totalErrors++
				}
				continue
			}
			sp.CheckPoint(icon.IconCheck, color.ColorGreen,
				fmt.Sprintf("  SSH connected to %s@%s", srv.User, srv.Host),
				color.ColorReset)
			defer sshClient.Close()
		}

		results := checkRule(rule, sshClient)
		for _, r := range results {
			switch r.status {
			case CheckOK:
				totalOK++
				fmt.Printf("  %s%s%s  %s  %s→%s  %s\n",
					color.ColorGreen, icon.IconCheck, color.ColorReset,
					r.item.Destination,
					color.ColorGray, color.ColorReset,
					r.detail)
			case CheckWrong:
				totalWrong++
				fmt.Printf("  %s%s%s  %s  %s→%s  %s\n",
					color.ColorYellow, icon.IconInfo, color.ColorReset,
					r.item.Destination,
					color.ColorGray, color.ColorReset,
					color.ColorYellow.Color(r.detail))
			case CheckMissing:
				totalMissing++
				fmt.Printf("  %s%s%s  %s  %s→%s  %s\n",
					color.ColorRed, icon.IconCross, color.ColorReset,
					r.item.Destination,
					color.ColorGray, color.ColorReset,
					color.ColorRed.Color(r.detail))
			case CheckError:
				totalErrors++
				fmt.Printf("  %s%s%s  %s  %s→%s  %s\n",
					color.ColorRed, icon.IconCross, color.ColorReset,
					r.item.Destination,
					color.ColorGray, color.ColorReset,
					color.ColorRed.Color(r.detail))
			}
		}
		fmt.Println()
	}

	// Summary line.
	sp.CheckPoint(icon.IconInfo, color.ColorYellow,
		fmt.Sprintf("Summary:  %s%d ✔ ok%s,  %s%d ! wrong%s,  %s%d ✘ missing%s,  %s%d errors%s.",
			color.ColorGreen, totalOK, color.ColorReset,
			color.ColorYellow, totalWrong, color.ColorReset,
			color.ColorRed, totalMissing, color.ColorReset,
			color.ColorRed, totalErrors, color.ColorReset),
		color.ColorReset)

	if totalWrong > 0 || totalMissing > 0 || totalErrors > 0 {
		os.Exit(1)
	}
}

// checkRule checks every item in rule and returns results.
// sshClient may be nil (used only when rule.Mode == ModeSSH).
func checkRule(rule *config.Rule, sshClient *sshutil.Client) []itemCheckResult {
	results := make([]itemCheckResult, 0, len(rule.Items))
	for _, item := range rule.Items {
		r := checkItem(item, rule.Mode, sshClient)
		results = append(results, r)
	}
	return results
}

// checkItem inspects a single item according to its deployment mode.
func checkItem(item *config.RuleItem, mode config.Mode, sshClient *sshutil.Client) itemCheckResult {
	switch mode {
	case config.ModeSymlink:
		return checkSymlink(item)
	case config.ModeCopy:
		return checkCopy(item)
	case config.ModeSSH:
		return checkSSH(item, sshClient)
	default:
		return checkSymlink(item)
	}
}

// checkSymlink verifies that dest is a symlink pointing to src.
func checkSymlink(item *config.RuleItem) itemCheckResult {
	fi, err := os.Lstat(item.Destination)
	if err != nil {
		if os.IsNotExist(err) {
			return itemCheckResult{item: item, status: CheckMissing, detail: "not found"}
		}
		return itemCheckResult{item: item, status: CheckError,
			detail: fmt.Sprintf("lstat error: %v", err)}
	}

	if fi.Mode()&os.ModeSymlink == 0 {
		return itemCheckResult{item: item, status: CheckWrong,
			detail: "exists but is not a symlink"}
	}

	target, err := os.Readlink(item.Destination)
	if err != nil {
		return itemCheckResult{item: item, status: CheckError,
			detail: fmt.Sprintf("readlink error: %v", err)}
	}

	if target != item.Source {
		return itemCheckResult{item: item, status: CheckWrong,
			detail: fmt.Sprintf("points to %s (expected %s)", target, item.Source)}
	}

	return itemCheckResult{item: item, status: CheckOK,
		detail: fmt.Sprintf("→  %s", item.Source)}
}

// checkCopy verifies that dest exists (copy mode doesn't track content equality).
func checkCopy(item *config.RuleItem) itemCheckResult {
	_, err := os.Stat(item.Destination)
	if err != nil {
		if os.IsNotExist(err) {
			return itemCheckResult{item: item, status: CheckMissing, detail: "not found"}
		}
		return itemCheckResult{item: item, status: CheckError,
			detail: fmt.Sprintf("stat error: %v", err)}
	}
	return itemCheckResult{item: item, status: CheckOK, detail: "exists"}
}

// checkSSH verifies that the remote path exists on the SSH server.
func checkSSH(item *config.RuleItem, client *sshutil.Client) itemCheckResult {
	if client == nil {
		return itemCheckResult{item: item, status: CheckError, detail: "no SSH connection"}
	}
	exists, err := client.Exists(item.Destination)
	if err != nil {
		return itemCheckResult{item: item, status: CheckError,
			detail: fmt.Sprintf("remote check error: %v", err)}
	}
	if !exists {
		return itemCheckResult{item: item, status: CheckMissing, detail: "not found on remote"}
	}
	return itemCheckResult{item: item, status: CheckOK, detail: "exists on remote"}
}
