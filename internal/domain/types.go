package domain

import "time"

type Mode string

const (
	ModeSymlink Mode = "symlink"
	ModeCopy    Mode = "copy"
	ModeSSH     Mode = "ssh"
)

type PathKind string

const (
	PathKindFile      PathKind = "file"
	PathKindDirectory PathKind = "directory"
)

type Hooks struct {
	Pre  string `yaml:"pre,omitempty" json:"pre,omitempty"`
	Post string `yaml:"post,omitempty" json:"post,omitempty"`
}

type SSHServer struct {
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port,omitempty" json:"port,omitempty"`
	User     string `yaml:"user" json:"user"`
	Key      string `yaml:"key,omitempty" json:"key,omitempty"`
	Password string `yaml:"password,omitempty" json:"password,omitempty"`
}

type RuleItem struct {
	Source      string   `yaml:"src" json:"src"`
	Destination string   `yaml:"dest" json:"dest"`
	Kind        PathKind `yaml:"-" json:"kind"`
}

type Rule struct {
	Name  string     `yaml:"name" json:"name"`
	Mode  Mode       `yaml:"mode,omitempty" json:"mode"`
	SSH   string     `yaml:"ssh,omitempty" json:"ssh,omitempty"`
	Hooks *Hooks     `yaml:"hooks,omitempty" json:"hooks,omitempty"`
	Items []RuleItem `yaml:"items" json:"items"`
}

type Config struct {
	ConfigPath  string               `yaml:"-" json:"config_path"`
	WorkDir     string               `yaml:"-" json:"work_dir"`
	Mode        Mode                 `yaml:"mode,omitempty" json:"mode,omitempty"`
	Hooks       *Hooks               `yaml:"hooks,omitempty" json:"hooks,omitempty"`
	SSHServers  map[string]SSHServer `yaml:"ssh_servers,omitempty" json:"ssh_servers,omitempty"`
	Vars        map[string]string    `yaml:"vars,omitempty" json:"vars,omitempty"`
	Rules       []Rule               `yaml:"rules" json:"rules"`
	SelectedRaw []string             `yaml:"-" json:"selected_rules,omitempty"`
}

type ActionType string

const (
	ActionRunHook       ActionType = "run_hook"
	ActionBackupLocal   ActionType = "backup_local"
	ActionBackupRemote  ActionType = "backup_remote"
	ActionDeploySymlink ActionType = "deploy_symlink"
	ActionDeployCopy    ActionType = "deploy_copy"
	ActionDeploySSH     ActionType = "deploy_ssh"
	ActionCheckSymlink  ActionType = "check_symlink"
	ActionCheckCopy     ActionType = "check_copy"
	ActionCheckSSH      ActionType = "check_ssh"
	ActionRestoreLocal  ActionType = "restore_local"
	ActionRestoreSSH    ActionType = "restore_ssh"
	ActionWriteManifest ActionType = "write_manifest"
)

type Action struct {
	Type         ActionType `json:"type"`
	RuleName     string     `json:"rule_name,omitempty"`
	Mode         Mode       `json:"mode,omitempty"`
	HookScope    string     `json:"hook_scope,omitempty"`
	HookCommand  string     `json:"hook_command,omitempty"`
	Source       string     `json:"source,omitempty"`
	Destination  string     `json:"destination,omitempty"`
	BackupPath   string     `json:"backup_path,omitempty"`
	SSHServer    string     `json:"ssh_server,omitempty"`
	PathKind     PathKind   `json:"path_kind,omitempty"`
	ExpectExists bool       `json:"expect_exists,omitempty"`
}

type Plan struct {
	Command         string    `json:"command"`
	ConfigPath      string    `json:"config_path,omitempty"`
	BackupDir       string    `json:"backup_dir,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	SelectedRules   []string  `json:"selected_rules,omitempty"`
	RequiredSecrets []string  `json:"required_secrets,omitempty"`
	Actions         []Action  `json:"actions"`
}

type BackupEntry struct {
	RuleName     string   `json:"rule_name"`
	Mode         Mode     `json:"mode"`
	Source       string   `json:"source"`
	Destination  string   `json:"destination"`
	PathKind     PathKind `json:"path_kind"`
	SSHServer    string   `json:"ssh_server,omitempty"`
	BackupPath   string   `json:"backup_path"`
	SHA256       string   `json:"sha256"`
	OriginalPath string   `json:"original_path"`
}

type BackupManifest struct {
	Version        int           `json:"version"`
	CreatedAt      time.Time     `json:"created_at"`
	Command        string        `json:"command"`
	ConfigPath     string        `json:"config_path"`
	ConfigSnapshot string        `json:"config_snapshot"`
	Entries        []BackupEntry `json:"entries"`
}

type CheckStatus string

const (
	CheckStatusOK      CheckStatus = "OK"
	CheckStatusDrifted CheckStatus = "DRIFTED"
	CheckStatusMissing CheckStatus = "MISSING"
	CheckStatusError   CheckStatus = "ERROR"
)

type ActionResult struct {
	Action      Action      `json:"action"`
	Status      string      `json:"status"`
	Detail      string      `json:"detail,omitempty"`
	CheckStatus CheckStatus `json:"check_status,omitempty"`
}

type ExecutionResult struct {
	Command  string         `json:"command"`
	Started  time.Time      `json:"started_at"`
	Finished time.Time      `json:"finished_at"`
	Success  int            `json:"success"`
	Skipped  int            `json:"skipped"`
	Failed   int            `json:"failed"`
	Results  []ActionResult `json:"results"`
}
