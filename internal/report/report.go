package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/alexmaze/clink/internal/domain"
)

type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

func PrintPlan(w io.Writer, format Format, plan *domain.Plan) error {
	if format == FormatJSON {
		return writeJSON(w, plan)
	}

	fmt.Fprintf(w, "Command: %s\n", plan.Command)
	if plan.ConfigPath != "" {
		fmt.Fprintf(w, "Config: %s\n", plan.ConfigPath)
	}
	if plan.BackupDir != "" {
		fmt.Fprintf(w, "Backup: %s\n", plan.BackupDir)
	}
	if len(plan.SelectedRules) > 0 {
		fmt.Fprintf(w, "Rules: %s\n", strings.Join(plan.SelectedRules, ", "))
	}
	if len(plan.RequiredSecrets) > 0 {
		fmt.Fprintf(w, "Secrets required: %s\n", strings.Join(plan.RequiredSecrets, ", "))
	}
	fmt.Fprintf(w, "Planned actions: %d\n", len(plan.Actions))
	for _, action := range plan.Actions {
		fmt.Fprintf(w, "  - %s", action.Type)
		if action.RuleName != "" {
			fmt.Fprintf(w, " [%s]", action.RuleName)
		}
		if action.Destination != "" {
			fmt.Fprintf(w, " %s", action.Destination)
		}
		fmt.Fprintln(w)
	}
	return nil
}

func PrintResult(w io.Writer, format Format, result *domain.ExecutionResult) error {
	if format == FormatJSON {
		return writeJSON(w, result)
	}

	fmt.Fprintf(w, "\nResult: %s\n", result.Command)
	fmt.Fprintf(w, "Success: %d  Skipped: %d  Failed: %d\n", result.Success, result.Skipped, result.Failed)
	for _, item := range result.Results {
		line := fmt.Sprintf("  - %s", item.Action.Type)
		if item.Action.RuleName != "" {
			line += " [" + item.Action.RuleName + "]"
		}
		if item.Action.Destination != "" {
			line += " " + item.Action.Destination
		}
		line += " => " + item.Status
		if item.CheckStatus != "" {
			line += " (" + string(item.CheckStatus) + ")"
		}
		if item.Detail != "" {
			line += ": " + item.Detail
		}
		fmt.Fprintln(w, line)
	}
	return nil
}

func writeJSON(w io.Writer, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
