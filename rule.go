package main

import "github.com/alexmaze/clink/lib/fileutil"

// Rule rule
type Rule struct {
	Name  string      `mapstructure:"name"`
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
