package config

import "github.com/alexmaze/clink/lib/fileutil"

// Mode defines the distribution mode for a rule
type Mode string

const (
	ModeSymlink Mode = "symlink"
	ModeCopy    Mode = "copy"
	ModeSSH     Mode = "ssh"
)

// SSHServer holds connection parameters for an SSH/SFTP target
type SSHServer struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`     // default 22
	User     string `mapstructure:"user"`
	Key      string `mapstructure:"key"`      // path to private key file (optional)
	Password string `mapstructure:"password"` // plaintext password (optional; empty → runtime prompt)
}

// Hooks holds pre/post shell commands for a rule or the global config
type Hooks struct {
	Pre  string `mapstructure:"pre"`
	Post string `mapstructure:"post"`
}

// Rule rule
type Rule struct {
	Name  string      `mapstructure:"name"`
	Mode  Mode        `mapstructure:"mode"` // optional; inherits from global if empty
	SSH   string      `mapstructure:"ssh"`  // key into ConfigFile.SSHServers (ssh mode only)
	Hooks *Hooks      `mapstructure:"hooks"`
	Items []*RuleItem `mapstructure:"items"`
}

// RuleItem files || folders
type RuleItem struct {
	Type        fileutil.PathType `mapstructure:"-"` // Auto detected by `Source`
	Source      string            `mapstructure:"src"`
	Destination string            `mapstructure:"dest"`
}
