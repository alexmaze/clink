package config

import "github.com/alexmaze/clink/lib/fileutil"

// Hooks holds pre/post shell commands for a rule or the global config
type Hooks struct {
	Pre  string `mapstructure:"pre"`
	Post string `mapstructure:"post"`
}

// Rule rule
type Rule struct {
	Name  string      `mapstructure:"name"`
	Hooks *Hooks      `mapstructure:"hooks"`
	Items []*RuleItem `mapstructure:"items"`
}

// Type configuration type
// type Type string

// // Types
// const (
// 	TypeFile   Type = "file"
// 	TypeFolder Type = "folder"
// )

// RuleItem files || folders
type RuleItem struct {
	Type        fileutil.PathType `mapstructure:"-"` // Auto detected by `Source`
	Source      string            `mapstructure:"src"`
	Destination string            `mapstructure:"dest"`
}
