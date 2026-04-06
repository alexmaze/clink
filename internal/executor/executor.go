package executor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/alexmaze/clink/internal/domain"
	"github.com/alexmaze/clink/lib/sshutil"
)

type Executor struct {
	Config   *domain.Config
	DryRun   bool
	Manifest *domain.BackupManifest
}

func New(cfg *domain.Config, dryRun bool) *Executor {
	return &Executor{
		Config: cfg,
		DryRun: dryRun,
		Manifest: &domain.BackupManifest{
			Version:    1,
			CreatedAt:  time.Now(),
			Command:    "apply",
			ConfigPath: cfg.ConfigPath,
		},
	}
}

func (e *Executor) Run(plan *domain.Plan) (*domain.ExecutionResult, error) {
	result := &domain.ExecutionResult{
		Command: plan.Command,
		Started: time.Now(),
	}

	clients := map[string]*sshutil.Client{}
	defer func() {
		for _, client := range clients {
			client.Close()
		}
	}()

	for _, action := range plan.Actions {
		itemResult := domain.ActionResult{Action: action}
		if e.DryRun {
			itemResult.Status = "DRY_RUN"
			result.Skipped++
			result.Results = append(result.Results, itemResult)
			continue
		}

		switch action.Type {
		case domain.ActionRunHook:
			err := runHook(action.HookCommand)
			if err != nil {
				itemResult.Status = "FAILED"
				itemResult.Detail = err.Error()
				result.Results = append(result.Results, itemResult)
				result.Failed++
				result.Finished = time.Now()
				return result, err
			}
			itemResult.Status = "OK"
			result.Success++
		case domain.ActionBackupLocal:
			status, detail, entry, err := e.backupLocal(action)
			itemResult.Status = status
			itemResult.Detail = detail
			if entry != nil {
				e.Manifest.Entries = append(e.Manifest.Entries, *entry)
			}
			countResult(result, status)
			if err != nil {
				result.Results = append(result.Results, itemResult)
				result.Finished = time.Now()
				return result, err
			}
		case domain.ActionBackupRemote:
			client, err := e.clientFor(clients, action.SSHServer)
			if err != nil {
				itemResult.Status = "FAILED"
				itemResult.Detail = err.Error()
				result.Results = append(result.Results, itemResult)
				result.Finished = time.Now()
				return result, err
			}
			status, detail, entry, err := backupRemote(client, action)
			itemResult.Status = status
			itemResult.Detail = detail
			if entry != nil {
				e.Manifest.Entries = append(e.Manifest.Entries, *entry)
			}
			countResult(result, status)
			if err != nil {
				result.Results = append(result.Results, itemResult)
				result.Finished = time.Now()
				return result, err
			}
		case domain.ActionDeploySymlink:
			err := deploySymlink(action)
			itemResult.Status = statusOf(err)
			itemResult.Detail = detailOf(err)
			countResult(result, itemResult.Status)
			if err != nil {
				result.Results = append(result.Results, itemResult)
				result.Finished = time.Now()
				return result, err
			}
		case domain.ActionDeployCopy:
			err := deployCopy(action)
			itemResult.Status = statusOf(err)
			itemResult.Detail = detailOf(err)
			countResult(result, itemResult.Status)
			if err != nil {
				result.Results = append(result.Results, itemResult)
				result.Finished = time.Now()
				return result, err
			}
		case domain.ActionDeploySSH:
			client, err := e.clientFor(clients, action.SSHServer)
			if err != nil {
				itemResult.Status = "FAILED"
				itemResult.Detail = err.Error()
				result.Results = append(result.Results, itemResult)
				result.Finished = time.Now()
				return result, err
			}
			err = client.Upload(action.Source, action.Destination)
			itemResult.Status = statusOf(err)
			itemResult.Detail = detailOf(err)
			countResult(result, itemResult.Status)
			if err != nil {
				result.Results = append(result.Results, itemResult)
				result.Finished = time.Now()
				return result, err
			}
		case domain.ActionCheckSymlink:
			itemResult.Status, itemResult.CheckStatus, itemResult.Detail = checkSymlink(action)
			countResult(result, itemResult.Status)
		case domain.ActionCheckCopy:
			itemResult.Status, itemResult.CheckStatus, itemResult.Detail = checkCopy(action)
			countResult(result, itemResult.Status)
		case domain.ActionCheckSSH:
			client, err := e.clientFor(clients, action.SSHServer)
			if err != nil {
				itemResult.Status = "FAILED"
				itemResult.CheckStatus = domain.CheckStatusError
				itemResult.Detail = err.Error()
				countResult(result, itemResult.Status)
				result.Results = append(result.Results, itemResult)
				result.Finished = time.Now()
				return result, err
			}
			itemResult.Status, itemResult.CheckStatus, itemResult.Detail = checkSSH(client, action)
			countResult(result, itemResult.Status)
		case domain.ActionRestoreLocal:
			err := restoreLocal(action)
			itemResult.Status = statusOf(err)
			itemResult.Detail = detailOf(err)
			countResult(result, itemResult.Status)
			if err != nil {
				result.Results = append(result.Results, itemResult)
				result.Finished = time.Now()
				return result, err
			}
		case domain.ActionRestoreSSH:
			client, err := e.clientFor(clients, action.SSHServer)
			if err != nil {
				itemResult.Status = "FAILED"
				itemResult.Detail = err.Error()
				result.Results = append(result.Results, itemResult)
				result.Finished = time.Now()
				return result, err
			}
			err = client.Upload(action.Source, action.Destination)
			itemResult.Status = statusOf(err)
			itemResult.Detail = detailOf(err)
			countResult(result, itemResult.Status)
			if err != nil {
				result.Results = append(result.Results, itemResult)
				result.Finished = time.Now()
				return result, err
			}
		case domain.ActionWriteManifest:
			e.Manifest.Command = plan.Command
			e.Manifest.ConfigSnapshot = filepath.Join(plan.BackupDir, "config.snapshot.yaml")
			err := writeManifest(action.BackupPath, e.Manifest)
			itemResult.Status = statusOf(err)
			itemResult.Detail = detailOf(err)
			countResult(result, itemResult.Status)
			if err != nil {
				result.Results = append(result.Results, itemResult)
				result.Finished = time.Now()
				return result, err
			}
		default:
			itemResult.Status = "SKIPPED"
			result.Skipped++
		}

		result.Results = append(result.Results, itemResult)
	}

	result.Finished = time.Now()
	return result, nil
}

func (e *Executor) clientFor(cache map[string]*sshutil.Client, name string) (*sshutil.Client, error) {
	if client, ok := cache[name]; ok {
		return client, nil
	}
	server, ok := e.Config.SSHServers[name]
	if !ok {
		return nil, fmt.Errorf("ssh server not found: %s", name)
	}
	client, err := sshutil.NewClient(&server)
	if err != nil {
		return nil, err
	}
	cache[name] = client
	return client, nil
}

func runHook(command string) error {
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (e *Executor) backupLocal(action domain.Action) (string, string, *domain.BackupEntry, error) {
	_, err := os.Lstat(action.Destination)
	if os.IsNotExist(err) {
		return "SKIPPED", "destination absent, backup skipped", nil, nil
	}
	if err != nil {
		return "FAILED", err.Error(), nil, err
	}
	if err := os.MkdirAll(filepath.Dir(action.BackupPath), 0755); err != nil {
		return "FAILED", err.Error(), nil, err
	}
	if err := copyPath(action.Destination, action.BackupPath); err != nil {
		return "FAILED", err.Error(), nil, err
	}
	sum, err := hashPath(action.BackupPath)
	if err != nil {
		return "FAILED", err.Error(), nil, err
	}
	entry := &domain.BackupEntry{
		RuleName:     action.RuleName,
		Mode:         action.Mode,
		Source:       action.Source,
		Destination:  action.Destination,
		PathKind:     action.PathKind,
		SSHServer:    action.SSHServer,
		BackupPath:   action.BackupPath,
		SHA256:       sum,
		OriginalPath: action.Destination,
	}
	return "OK", "backup captured", entry, nil
}

func backupRemote(client *sshutil.Client, action domain.Action) (string, string, *domain.BackupEntry, error) {
	exists, err := client.Exists(action.Destination)
	if err != nil {
		return "FAILED", err.Error(), nil, err
	}
	if !exists {
		return "SKIPPED", "remote destination absent, backup skipped", nil, nil
	}
	if err := os.MkdirAll(filepath.Dir(action.BackupPath), 0755); err != nil {
		return "FAILED", err.Error(), nil, err
	}
	if err := client.Download(action.Destination, action.BackupPath); err != nil {
		return "FAILED", err.Error(), nil, err
	}
	sum, err := hashPath(action.BackupPath)
	if err != nil {
		return "FAILED", err.Error(), nil, err
	}
	entry := &domain.BackupEntry{
		RuleName:     action.RuleName,
		Mode:         action.Mode,
		Source:       action.Source,
		Destination:  action.Destination,
		PathKind:     action.PathKind,
		SSHServer:    action.SSHServer,
		BackupPath:   action.BackupPath,
		SHA256:       sum,
		OriginalPath: action.Destination,
	}
	return "OK", "backup captured", entry, nil
}

func deploySymlink(action domain.Action) error {
	if err := os.MkdirAll(filepath.Dir(action.Destination), 0755); err != nil {
		return err
	}
	if err := os.RemoveAll(action.Destination); err != nil {
		return err
	}
	return os.Symlink(action.Source, action.Destination)
}

func deployCopy(action domain.Action) error {
	if err := os.MkdirAll(filepath.Dir(action.Destination), 0755); err != nil {
		return err
	}
	if err := os.RemoveAll(action.Destination); err != nil {
		return err
	}
	return copyPath(action.Source, action.Destination)
}

func restoreLocal(action domain.Action) error {
	tmp := action.Destination + ".clink-restore-tmp"
	if err := os.MkdirAll(filepath.Dir(action.Destination), 0755); err != nil {
		return err
	}
	if err := os.RemoveAll(tmp); err != nil {
		return err
	}
	if err := copyPath(action.Source, tmp); err != nil {
		return err
	}
	if err := os.RemoveAll(action.Destination); err != nil {
		return err
	}
	return os.Rename(tmp, action.Destination)
}

func writeManifest(path string, manifest *domain.BackupManifest) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, body, 0644)
}

func checkSymlink(action domain.Action) (string, domain.CheckStatus, string) {
	info, err := os.Lstat(action.Destination)
	if os.IsNotExist(err) {
		return "FAILED", domain.CheckStatusMissing, "destination missing"
	}
	if err != nil {
		return "FAILED", domain.CheckStatusError, err.Error()
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return "FAILED", domain.CheckStatusDrifted, "destination is not a symlink"
	}
	target, err := os.Readlink(action.Destination)
	if err != nil {
		return "FAILED", domain.CheckStatusError, err.Error()
	}
	if target != action.Source {
		return "FAILED", domain.CheckStatusDrifted, fmt.Sprintf("symlink target mismatch: %s", target)
	}
	return "OK", domain.CheckStatusOK, "symlink target matches"
}

func checkCopy(action domain.Action) (string, domain.CheckStatus, string) {
	_, err := os.Stat(action.Destination)
	if os.IsNotExist(err) {
		return "FAILED", domain.CheckStatusMissing, "destination missing"
	}
	if err != nil {
		return "FAILED", domain.CheckStatusError, err.Error()
	}
	srcHash, err := hashPath(action.Source)
	if err != nil {
		return "FAILED", domain.CheckStatusError, err.Error()
	}
	destHash, err := hashPath(action.Destination)
	if err != nil {
		return "FAILED", domain.CheckStatusError, err.Error()
	}
	if srcHash != destHash {
		return "FAILED", domain.CheckStatusDrifted, "content hash mismatch"
	}
	return "OK", domain.CheckStatusOK, "content matches"
}

func checkSSH(client *sshutil.Client, action domain.Action) (string, domain.CheckStatus, string) {
	info, err := client.Stat(action.Destination)
	if err != nil {
		if os.IsNotExist(err) {
			return "FAILED", domain.CheckStatusMissing, "remote destination missing"
		}
		return "FAILED", domain.CheckStatusError, err.Error()
	}
	if action.PathKind == domain.PathKindDirectory && !info.IsDir() {
		return "FAILED", domain.CheckStatusDrifted, "remote destination type mismatch"
	}
	if action.PathKind == domain.PathKindFile && info.IsDir() {
		return "FAILED", domain.CheckStatusDrifted, "remote destination type mismatch"
	}
	return "OK", domain.CheckStatusOK, "remote destination exists"
}

func copyPath(src, dest string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		_ = os.RemoveAll(dest)
		return os.Symlink(target, dest)
	}
	if !info.IsDir() {
		return copyFile(src, dest, info.Mode())
	}
	if err := os.MkdirAll(dest, info.Mode()); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := copyPath(filepath.Join(src, entry.Name()), filepath.Join(dest, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dest string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(dest, mode)
}

func hashPath(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256([]byte("symlink:" + target))
		return hex.EncodeToString(sum[:]), nil
	}
	if !info.IsDir() {
		return hashFile(path)
	}

	h := sha256.New()
	entries := []string{}
	if err := filepath.Walk(path, func(current string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if current == path {
			return nil
		}
		rel, err := filepath.Rel(path, current)
		if err != nil {
			return err
		}
		entries = append(entries, rel)
		return nil
	}); err != nil {
		return "", err
	}
	sort.Strings(entries)
	for _, rel := range entries {
		full := filepath.Join(path, rel)
		info, err := os.Lstat(full)
		if err != nil {
			return "", err
		}
		io.WriteString(h, rel)
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(full)
			if err != nil {
				return "", err
			}
			io.WriteString(h, "symlink")
			io.WriteString(h, target)
			continue
		}
		if info.IsDir() {
			io.WriteString(h, "dir")
			continue
		}
		sum, err := hashFile(full)
		if err != nil {
			return "", err
		}
		io.WriteString(h, sum)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func statusOf(err error) string {
	if err != nil {
		return "FAILED"
	}
	return "OK"
}

func detailOf(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func countResult(result *domain.ExecutionResult, status string) {
	switch strings.ToUpper(status) {
	case "OK":
		result.Success++
	case "SKIPPED", "DRY_RUN":
		result.Skipped++
	default:
		result.Failed++
	}
}
