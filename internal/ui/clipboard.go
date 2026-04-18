package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// CopyToClipboard copies text to the system clipboard.
func CopyToClipboard(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, then xsel, then wl-copy (Wayland)
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		} else {
			return fmt.Errorf("no clipboard tool found (install xclip, xsel, or wl-copy)")
		}
	case "windows":
		cmd = exec.Command("cmd", "/c", "clip")
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// CopyCellValue copies a single cell value.
func CopyCellValue(columns []string, row []string, colIdx int) (string, error) {
	if colIdx >= len(row) {
		return "", fmt.Errorf("column index out of range")
	}
	val := row[colIdx]
	if val == "<NULL>" {
		val = "NULL"
	}
	return val, CopyToClipboard(val)
}

// CopyRowAsJSON copies a row as a JSON object.
func CopyRowAsJSON(columns []string, row []string) (string, error) {
	var b strings.Builder
	b.WriteString("{\n")
	for i, col := range columns {
		val := ""
		if i < len(row) {
			val = row[i]
		}
		if val == "<NULL>" {
			b.WriteString(fmt.Sprintf("  %q: null", col))
		} else {
			b.WriteString(fmt.Sprintf("  %q: %q", col, val))
		}
		if i < len(columns)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("}")
	text := b.String()
	return text, CopyToClipboard(text)
}

// CopyRowAsTSV copies a row as tab-separated values.
func CopyRowAsTSV(row []string) (string, error) {
	text := strings.Join(row, "\t")
	return text, CopyToClipboard(text)
}

// CopyResultsAsCSV copies all results as CSV.
func CopyResultsAsCSV(columns []string, rows [][]string) (string, error) {
	var b strings.Builder
	b.WriteString(strings.Join(columns, ",") + "\n")
	for _, row := range rows {
		var escaped []string
		for _, cell := range row {
			if strings.ContainsAny(cell, ",\"\n") {
				cell = `"` + strings.ReplaceAll(cell, `"`, `""`) + `"`
			}
			escaped = append(escaped, cell)
		}
		b.WriteString(strings.Join(escaped, ",") + "\n")
	}
	text := b.String()
	return text, CopyToClipboard(text)
}
