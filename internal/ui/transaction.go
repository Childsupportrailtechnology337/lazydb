package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// TransactionState represents the current transaction state.
type TransactionState int

const (
	TxNone TransactionState = iota
	TxActive
	TxPendingCommit
	TxPendingRollback
)

// TransactionMsg is sent for transaction control.
type TransactionMsg struct {
	Action string // "begin", "commit", "rollback"
}

// TransactionIndicator shows the current transaction state in the UI.
type TransactionIndicator struct {
	state    TransactionState
	queries  int // Number of queries in current transaction
}

// NewTransactionIndicator creates a new transaction indicator.
func NewTransactionIndicator() TransactionIndicator {
	return TransactionIndicator{state: TxNone}
}

// State returns the current transaction state.
func (t *TransactionIndicator) State() TransactionState {
	return t.state
}

// Begin starts a transaction.
func (t *TransactionIndicator) Begin() {
	t.state = TxActive
	t.queries = 0
}

// AddQuery increments the query count.
func (t *TransactionIndicator) AddQuery() {
	if t.state == TxActive {
		t.queries++
	}
}

// Commit marks the transaction as committed.
func (t *TransactionIndicator) Commit() {
	t.state = TxNone
	t.queries = 0
}

// Rollback marks the transaction as rolled back.
func (t *TransactionIndicator) Rollback() {
	t.state = TxNone
	t.queries = 0
}

// IsActive returns whether a transaction is active.
func (t *TransactionIndicator) IsActive() bool {
	return t.state == TxActive
}

// QueryCount returns the number of queries in the current transaction.
func (t *TransactionIndicator) QueryCount() int {
	return t.queries
}

// View renders the transaction indicator for the status bar.
func (t *TransactionIndicator) View() string {
	switch t.state {
	case TxActive:
		style := lipgloss.NewStyle().
			Background(lipgloss.Color("#F59E0B")).
			Foreground(lipgloss.Color("#000000")).
			Bold(true).
			Padding(0, 1)

		var b strings.Builder
		b.WriteString(" TX ACTIVE")
		if t.queries > 0 {
			b.WriteString(" (")
			b.WriteString(intToStr(t.queries))
			b.WriteString(" queries)")
		}
		return style.Render(b.String())
	default:
		return ""
	}
}

// ConfirmDialog shows a confirmation for commit/rollback.
type TxConfirmDialog struct {
	visible bool
	action  string // "commit" or "rollback"
	width   int
	height  int
}

// NewTxConfirmDialog creates a new transaction confirmation dialog.
func NewTxConfirmDialog() TxConfirmDialog {
	return TxConfirmDialog{}
}

// Show shows the confirmation dialog.
func (d *TxConfirmDialog) Show(action string) {
	d.visible = true
	d.action = action
}

// Hide hides the dialog.
func (d *TxConfirmDialog) Hide() {
	d.visible = false
}

// IsVisible returns whether the dialog is visible.
func (d *TxConfirmDialog) IsVisible() bool {
	return d.visible
}

// Action returns the pending action.
func (d *TxConfirmDialog) Action() string {
	return d.action
}

// SetSize sets the dialog dimensions.
func (d *TxConfirmDialog) SetSize(w, h int) {
	d.width = w
	d.height = h
}

// View renders the confirmation dialog.
func (d *TxConfirmDialog) View() string {
	if !d.visible {
		return ""
	}

	var b strings.Builder
	title := lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Render("  Transaction")
	b.WriteString(title + "\n\n")

	if d.action == "commit" {
		b.WriteString("  " + lipgloss.NewStyle().Foreground(SuccessColor).Bold(true).Render("COMMIT") + " this transaction?\n\n")
	} else {
		b.WriteString("  " + lipgloss.NewStyle().Foreground(ErrorColor).Bold(true).Render("ROLLBACK") + " this transaction?\n\n")
		b.WriteString("  " + lipgloss.NewStyle().Foreground(ErrorColor).Render("All changes will be lost.") + "\n\n")
	}

	b.WriteString(MutedText.Render("  [Y] Confirm  [N] Cancel"))

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Width(40).
		Padding(1, 2).
		Render(b.String())

	return lipgloss.Place(d.width, d.height, lipgloss.Center, lipgloss.Center, modal)
}
