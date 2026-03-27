package config

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

var (
	tableHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	tableGrayStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// RenderRulesTable returns a formatted string showing:
//  1. A summary table: # | Rule | Mode | Items
//  2. Per-rule expanded item details (hooks + items)
func RenderRulesTable(configFile *ConfigFile) string {
	var sb strings.Builder

	type row struct {
		idx   string
		name  string
		mode  string
		items string
	}

	rows := make([]row, len(configFile.Rules))
	for i, rule := range configFile.Rules {
		rows[i] = row{
			idx:   fmt.Sprintf("%d", i+1),
			name:  rule.Name,
			mode:  BuildModeLabel(configFile, rule),
			items: fmt.Sprintf("%d", len(rule.Items)),
		}
	}

	// Column widths (minimum: header lengths)
	wIdx := runewidth.StringWidth("#")
	wName := runewidth.StringWidth("Rule")
	wMode := runewidth.StringWidth("Mode")
	wItems := runewidth.StringWidth("Items")

	for _, r := range rows {
		if n := runewidth.StringWidth(r.idx); n > wIdx {
			wIdx = n
		}
		if n := runewidth.StringWidth(r.name); n > wName {
			wName = n
		}
		if n := runewidth.StringWidth(r.mode); n > wMode {
			wMode = n
		}
		if n := runewidth.StringWidth(r.items); n > wItems {
			wItems = n
		}
	}

	// Helpers
	hline := func(left, cross, right string, ws ...int) string {
		parts := make([]string, len(ws))
		for i, w := range ws {
			parts[i] = strings.Repeat("─", w+2)
		}
		return left + strings.Join(parts, cross) + right
	}

	pad := func(s string, w int, rightAlign bool) string {
		n := runewidth.StringWidth(s)
		p := w - n
		if p < 0 {
			p = 0
		}
		if rightAlign {
			return strings.Repeat(" ", p) + s
		}
		return s + strings.Repeat(" ", p)
	}

	top := hline("┌", "┬", "┐", wIdx, wName, wMode, wItems)
	sep := hline("├", "┼", "┤", wIdx, wName, wMode, wItems)
	bot := hline("└", "┴", "┘", wIdx, wName, wMode, wItems)

	fmtRow := func(idx, name, mode, items string, header bool) string {
		idxCell := pad(idx, wIdx, false)
		nameCell := pad(name, wName, false)
		modeCell := pad(mode, wMode, false)
		itemsCell := pad(items, wItems, true)
		if header {
			idxCell = tableHeaderStyle.Render(idxCell)
			nameCell = tableHeaderStyle.Render(nameCell)
			modeCell = tableHeaderStyle.Render(modeCell)
			itemsCell = tableHeaderStyle.Render(itemsCell)
		} else {
			idxCell = tableGrayStyle.Render(idxCell)
			itemsCell = tableGrayStyle.Render(itemsCell)
		}
		return fmt.Sprintf("│ %s │ %s │ %s │ %s │",
			idxCell, nameCell, modeCell, itemsCell)
	}

	sb.WriteString("  " + top + "\n")
	sb.WriteString("  " + fmtRow("#", "Rule", "Mode", "Items", true) + "\n")
	sb.WriteString("  " + sep + "\n")
	for _, r := range rows {
		sb.WriteString("  " + fmtRow(r.idx, r.name, r.mode, r.items, false) + "\n")
	}
	sb.WriteString("  " + bot + "\n\n")

	// Expanded per-rule details
	for i, rule := range configFile.Rules {
		modeLabel := BuildModeLabel(configFile, rule)
		sb.WriteString(fmt.Sprintf("  \033[36m[%d]\033[0m %s  \033[90m[%s]\033[0m\n",
			i+1, rule.Name, modeLabel))

		if rule.Hooks != nil {
			if rule.Hooks.Pre != "" {
				sb.WriteString(fmt.Sprintf("      \033[33mpre-hook:\033[0m  %s\n", rule.Hooks.Pre))
			}
			if rule.Hooks.Post != "" {
				sb.WriteString(fmt.Sprintf("      \033[33mpost-hook:\033[0m %s\n", rule.Hooks.Post))
			}
		}
		for _, item := range rule.Items {
			sb.WriteString(fmt.Sprintf("      • %s  \033[90m[%s]\033[0m\n        \033[90m→\033[0m  %s\n",
				item.Source, string(item.Type), item.Destination))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
