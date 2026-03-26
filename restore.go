package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/alexmaze/clink/config"
	"github.com/alexmaze/clink/lib/color"
	"github.com/alexmaze/clink/lib/fileutil"
	"github.com/alexmaze/clink/lib/icon"
	"github.com/alexmaze/clink/lib/spinner"
	"github.com/alexmaze/clink/lib/sshutil"
	"github.com/manifoldco/promptui"
)

// BackupEntry represents a single timestamped backup directory under ~/.clink/.
type BackupEntry struct {
	Path      string    // absolute path to the backup directory
	Timestamp time.Time // parsed from directory name
	HasConfig bool      // whether config.yaml snapshot exists
	FileCount int       // number of backed-up files (excluding config.yaml)
}

// RestoreItem represents one file to be restored from a backup.
type RestoreItem struct {
	BackupFile  string      // absolute path to the backup file
	Destination string      // original target path
	Mode        config.Mode // symlink/copy/ssh
	SSHServer   string      // SSH server key (ssh mode only)
	RuleName    string      // which rule this item belongs to
}

var backupDirPattern = regexp.MustCompile(`^\d{8}_\d{6}$`)

// runRestore is the entry point for `clink --restore`.
func runRestore(opts ClinkOpts) {
	sp := spinner.New()

	// 1. Locate ~/.clink/
	clinkDir, err := fileutil.ParsePath("", "~/.clink")
	if err != nil {
		sp.Failedf("Failed to resolve ~/.clink: %v", err)
		os.Exit(1)
	}

	// 2. Scan backups
	backups, err := scanBackups(clinkDir)
	if err != nil {
		sp.Failedf("Failed to scan backups: %v", err)
		os.Exit(1)
	}
	if len(backups) == 0 {
		sp.Failed("No backups found in " + clinkDir)
		os.Exit(0)
	}

	sp.CheckPoint(icon.IconInfo, color.ColorCyan,
		fmt.Sprintf("Found %d backup(s) in %s", len(backups), clinkDir),
		color.ColorReset)

	// 3. Let user pick a backup
	selected, err := promptBackupSelection(backups)
	if err != nil {
		sp.Failed("Canceled")
		os.Exit(0)
	}

	sp.CheckPoint(icon.IconCheck, color.ColorGreen,
		fmt.Sprintf("Selected: %s", selected.Timestamp.Format("2006-01-02 15:04:05")),
		color.ColorReset)

	// 4. Parse config snapshot (if available)
	var configFile *config.ConfigFile
	if selected.HasConfig {
		cfgPath := filepath.Join(selected.Path, "config.yaml")
		configFile, err = config.ParseConfigFileOnly(cfgPath)
		if err != nil {
			sp.CheckPoint(icon.IconInfo, color.ColorYellow,
				fmt.Sprintf("Warning: failed to parse config snapshot: %v", err),
				color.ColorReset)
			sp.CheckPoint(icon.IconInfo, color.ColorYellow,
				"All files will be restored as copy mode to local paths.",
				color.ColorReset)
			configFile = nil
		}
	} else {
		sp.CheckPoint(icon.IconInfo, color.ColorYellow,
			"No config.yaml snapshot in this backup. All files will be restored as copy mode to local paths.",
			color.ColorReset)
	}

	// 5. Build restore plan
	items, err := buildRestorePlan(selected, configFile)
	if err != nil {
		sp.Failedf("Failed to build restore plan: %v", err)
		os.Exit(1)
	}
	if len(items) == 0 {
		sp.Failed("No files to restore in this backup.")
		os.Exit(0)
	}

	// 6. Filter by -r if specified
	if len(opts.Rules) > 0 {
		items = filterRestoreItems(sp, items, opts.Rules)
		if len(items) == 0 {
			sp.Failed("No files match the specified rule filter.")
			os.Exit(0)
		}
	}

	// 7. Preview
	printRestorePlan(sp, items)

	if opts.DryRun {
		fmt.Println()
		sp.CheckPoint(icon.IconInfo, color.ColorYellow,
			"[dry-run] No files were restored.", color.ColorReset)
		return
	}

	// 8. Confirm
	p := promptui.Prompt{
		Label:     "Proceed with restore",
		IsConfirm: true,
	}
	if _, confirmErr := p.Run(); confirmErr != nil {
		sp.Failed("Canceled")
		os.Exit(0)
	}

	// 9. Execute
	restored, failed := executeRestore(sp, items, configFile)

	// 10. Summary
	fmt.Println()
	sp.Successf("Done! %s%d restored%s, %s%d failed%s.",
		color.ColorGreen, restored, color.ColorReset,
		color.ColorRed, failed, color.ColorReset)
}

// scanBackups reads ~/.clink/ and returns backup entries sorted by time descending.
func scanBackups(clinkDir string) ([]BackupEntry, error) {
	entries, err := os.ReadDir(clinkDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var backups []BackupEntry
	for _, entry := range entries {
		if !entry.IsDir() || !backupDirPattern.MatchString(entry.Name()) {
			continue
		}

		ts, err := time.ParseInLocation("20060102_150405", entry.Name(), time.Local)
		if err != nil {
			continue
		}

		dirPath := filepath.Join(clinkDir, entry.Name())
		hasConfig := false
		if _, statErr := os.Stat(filepath.Join(dirPath, "config.yaml")); statErr == nil {
			hasConfig = true
		}

		fileCount := countFiles(dirPath, hasConfig)
		if fileCount == 0 {
			continue // skip empty backups
		}

		backups = append(backups, BackupEntry{
			Path:      dirPath,
			Timestamp: ts,
			HasConfig: hasConfig,
			FileCount: fileCount,
		})
	}

	// Sort by time descending (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// countFiles counts files in a backup directory, optionally excluding config.yaml.
func countFiles(dir string, hasConfig bool) int {
	count := 0
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		// Exclude the config snapshot itself from the file count
		if hasConfig && path == filepath.Join(dir, "config.yaml") {
			return nil
		}
		count++
		return nil
	})
	return count
}

// promptBackupSelection presents a promptui.Select for the user to pick a backup.
func promptBackupSelection(backups []BackupEntry) (BackupEntry, error) {
	items := make([]string, len(backups))
	for i, b := range backups {
		configTag := "config ✘"
		if b.HasConfig {
			configTag = "config ✔"
		}
		items[i] = fmt.Sprintf("%s  (%d files, %s)",
			b.Timestamp.Format("2006-01-02 15:04:05"),
			b.FileCount, configTag)
	}

	sel := promptui.Select{
		Label: "Select a backup to restore",
		Items: items,
		Size:  10,
	}

	idx, _, err := sel.Run()
	if err != nil {
		return BackupEntry{}, err
	}
	return backups[idx], nil
}

// buildRestorePlan creates a list of RestoreItems by cross-referencing backup
// files with the config snapshot. If configFile is nil, all files are treated
// as copy-mode local restores.
func buildRestorePlan(backup BackupEntry, configFile *config.ConfigFile) ([]RestoreItem, error) {
	var items []RestoreItem

	// Collect all files in the backup (excluding config.yaml)
	var backupFiles []string
	configYamlPath := filepath.Join(backup.Path, "config.yaml")
	err := filepath.Walk(backup.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if path == configYamlPath {
			return nil
		}
		backupFiles = append(backupFiles, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if configFile == nil {
		// No config snapshot: treat all as copy-mode local restore.
		// The destination is the relative path under the backup dir interpreted as
		// an absolute path. Backup structure mirrors the destination paths.
		for _, bf := range backupFiles {
			relPath, _ := filepath.Rel(backup.Path, bf)
			dest := "/" + relPath
			items = append(items, RestoreItem{
				BackupFile:  bf,
				Destination: dest,
				Mode:        config.ModeCopy,
				RuleName:    "(unknown)",
			})
		}
		return items, nil
	}

	// With config: match backup files to rules
	// Build a lookup: destination → (rule, item)
	type ruleRef struct {
		ruleName  string
		mode      config.Mode
		sshServer string
		dest      string
	}
	destLookup := map[string]ruleRef{}

	for _, rule := range configFile.Rules {
		for _, item := range rule.Items {
			destLookup[item.Destination] = ruleRef{
				ruleName:  rule.Name,
				mode:      rule.Mode,
				sshServer: rule.SSH,
				dest:      item.Destination,
			}
		}
	}

	for _, bf := range backupFiles {
		relPath, _ := filepath.Rel(backup.Path, bf)
		// The backup file path structure mirrors the destination path
		dest := "/" + relPath

		ref, found := destLookup[dest]
		if found {
			items = append(items, RestoreItem{
				BackupFile:  bf,
				Destination: ref.dest,
				Mode:        ref.mode,
				SSHServer:   ref.sshServer,
				RuleName:    ref.ruleName,
			})
		} else {
			// File exists in backup but doesn't match any rule dest —
			// could be a sub-file of a directory item. Try prefix match.
			matched := false
			for destPath, r := range destLookup {
				if strings.HasPrefix(dest, destPath+"/") {
					items = append(items, RestoreItem{
						BackupFile:  bf,
						Destination: dest,
						Mode:        r.mode,
						SSHServer:   r.sshServer,
						RuleName:    r.ruleName,
					})
					matched = true
					break
				}
			}
			if !matched {
				// Fallback: restore as copy-mode local
				items = append(items, RestoreItem{
					BackupFile:  bf,
					Destination: dest,
					Mode:        config.ModeCopy,
					RuleName:    "(unmatched)",
				})
			}
		}
	}

	return items, nil
}

// filterRestoreItems filters items by rule name or 1-based rule index.
func filterRestoreItems(sp spinner.Spinner, items []RestoreItem, filters []string) []RestoreItem {
	// Collect unique rule names in order
	var ruleNames []string
	seen := map[string]bool{}
	for _, item := range items {
		if !seen[item.RuleName] {
			seen[item.RuleName] = true
			ruleNames = append(ruleNames, item.RuleName)
		}
	}

	// Build the set of matching rule names
	matchNames := map[string]bool{}
	for _, token := range filters {
		matched := false
		// Try as 1-based index
		if n := 0; true {
			fmt.Sscanf(token, "%d", &n)
			if n >= 1 && n <= len(ruleNames) {
				matchNames[ruleNames[n-1]] = true
				matched = true
			}
		}
		// Try as name (case-insensitive)
		for _, name := range ruleNames {
			if strings.EqualFold(name, token) {
				matchNames[name] = true
				matched = true
			}
		}
		if !matched {
			sp.CheckPoint(icon.IconInfo, color.ColorYellow,
				fmt.Sprintf("rule not found: %q", token), color.ColorReset)
		}
	}

	var filtered []RestoreItem
	for _, item := range items {
		if matchNames[item.RuleName] {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// printRestorePlan displays the restore plan grouped by rule.
func printRestorePlan(sp spinner.Spinner, items []RestoreItem) {
	// Group by rule name preserving order
	type ruleGroup struct {
		name  string
		items []RestoreItem
	}
	var groups []ruleGroup
	groupIdx := map[string]int{}

	for _, item := range items {
		idx, ok := groupIdx[item.RuleName]
		if !ok {
			idx = len(groups)
			groupIdx[item.RuleName] = idx
			groups = append(groups, ruleGroup{name: item.RuleName})
		}
		groups[idx].items = append(groups[idx].items, item)
	}

	fmt.Println()
	sp.CheckPoint(icon.IconInfo, color.ColorYellow,
		fmt.Sprintf("Restore plan: (%d files)", len(items)),
		color.ColorReset)
	fmt.Println()

	for i, g := range groups {
		// Build mode label from first item
		modeLabel := string(g.items[0].Mode)
		if g.items[0].Mode == config.ModeSSH && g.items[0].SSHServer != "" {
			modeLabel = fmt.Sprintf("ssh → %s", g.items[0].SSHServer)
		}

		fmt.Printf("  %s[%d]%s %s  %s[%s]%s\n",
			color.ColorCyan, i+1, color.ColorReset,
			color.ColorWhite.Color(g.name),
			color.ColorGray, modeLabel, color.ColorReset)

		for _, item := range g.items {
			fmt.Printf("      • %s\n        %s→%s  %s\n",
				item.BackupFile,
				color.ColorGray, color.ColorReset,
				item.Destination)
		}
		fmt.Println()
	}
}

// executeRestore restores all items and returns counts of successes and failures.
func executeRestore(sp spinner.Spinner, items []RestoreItem, configFile *config.ConfigFile) (restored, failed int) {
	// Group SSH items by server key so we can reuse connections
	type sshGroup struct {
		serverKey string
		items     []RestoreItem
	}
	var sshGroups []sshGroup
	sshGroupIdx := map[string]int{}

	var localItems []RestoreItem

	for _, item := range items {
		if item.Mode == config.ModeSSH {
			idx, ok := sshGroupIdx[item.SSHServer]
			if !ok {
				idx = len(sshGroups)
				sshGroupIdx[item.SSHServer] = idx
				sshGroups = append(sshGroups, sshGroup{serverKey: item.SSHServer})
			}
			sshGroups[idx].items = append(sshGroups[idx].items, item)
		} else {
			localItems = append(localItems, item)
		}
	}

	// Restore local items (symlink/copy → always use copy)
	for _, item := range localItems {
		sp.Start(fmt.Sprintf("Restoring %s ...", item.Destination))

		if err := restoreLocal(item); err != nil {
			sp.Stop()
			sp.CheckPoint(icon.IconCross, color.ColorRed,
				fmt.Sprintf("  → restore  %s  failed: %s", item.Destination, err.Error()),
				color.ColorReset)
			failed++
			continue
		}

		sp.Stop()
		sp.CheckPoint(icon.IconCheck, color.ColorGreen,
			fmt.Sprintf("  → restore  %s  ✔", item.Destination),
			color.ColorReset)
		restored++
	}

	// Restore SSH items
	for _, sg := range sshGroups {
		if configFile == nil || configFile.SSHServers == nil {
			for range sg.items {
				sp.CheckPoint(icon.IconCross, color.ColorRed,
					fmt.Sprintf("  → SSH server %q not available (no config snapshot)", sg.serverKey),
					color.ColorReset)
				failed++
			}
			continue
		}

		srv, ok := configFile.SSHServers[sg.serverKey]
		if !ok {
			for range sg.items {
				sp.CheckPoint(icon.IconCross, color.ColorRed,
					fmt.Sprintf("  → SSH server %q not found in config", sg.serverKey),
					color.ColorReset)
				failed++
			}
			continue
		}

		// Prompt password if needed
		if srv.Key == "" && srv.Password == "" {
			p := promptui.Prompt{
				Label: fmt.Sprintf("Password for %s@%s (server: %s)", srv.User, srv.Host, sg.serverKey),
				Mask:  '*',
			}
			pwd, err := p.Run()
			if err != nil {
				sp.Failedf("Failed to read password for server %s: %v", sg.serverKey, err)
				for range sg.items {
					failed++
				}
				continue
			}
			srv.Password = pwd
		}

		sp.Start(fmt.Sprintf("Connecting to %s@%s ...", srv.User, srv.Host))
		client, err := sshutil.NewClient(srv)
		sp.Stop()
		if err != nil {
			sp.CheckPoint(icon.IconCross, color.ColorRed,
				fmt.Sprintf("  → SSH connect failed: %s", err.Error()),
				color.ColorReset)

			// Ask whether to continue
			cp := promptui.Prompt{
				Label:     "Continue anyway",
				IsConfirm: true,
			}
			if _, confirmErr := cp.Run(); confirmErr != nil {
				sp.Failed("Aborted")
				os.Exit(0)
			}
			for range sg.items {
				failed++
			}
			continue
		}

		sp.CheckPoint(icon.IconCheck, color.ColorGreen,
			fmt.Sprintf("  → SSH connected to %s@%s", srv.User, srv.Host),
			color.ColorReset)

		for _, item := range sg.items {
			sp.Start(fmt.Sprintf("Uploading %s ...", item.Destination))

			if uploadErr := client.Upload(item.BackupFile, item.Destination); uploadErr != nil {
				sp.Stop()
				sp.CheckPoint(icon.IconCross, color.ColorRed,
					fmt.Sprintf("  → restore  %s  failed: %s", item.Destination, uploadErr.Error()),
					color.ColorReset)
				failed++
				continue
			}

			sp.Stop()
			sp.CheckPoint(icon.IconCheck, color.ColorGreen,
				fmt.Sprintf("  → restore  %s  ✔", item.Destination),
				color.ColorReset)
			restored++
		}

		client.Close()
	}

	return restored, failed
}

// restoreLocal restores a single file locally by copying from the backup.
func restoreLocal(item RestoreItem) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(item.Destination), 0755); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}

	// Remove existing destination (could be symlink, file, or directory)
	if err := os.RemoveAll(item.Destination); err != nil {
		return fmt.Errorf("remove existing: %w", err)
	}

	return copyPath(item.BackupFile, item.Destination)
}
