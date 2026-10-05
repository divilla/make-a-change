package documents

import (
	"cli/internal/dto"
	"cli/internal/styles"
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Printer captures full colored Markdown output without using a shell.
type Printer interface {
	Print(context.Context, string) (string, error)
}

// History owns retained rows, stable selection, viewport and operation lifetime.
type History struct {
	ProjectID, OwnerID        int
	Table, Type               string
	Rows, Active              []dto.Document
	Selected, Offset          int
	Revision                  uint64
	Busy                      bool
	Output, Status, Committed string
	Err                       error
	cancel                    context.CancelFunc
	mutating                  bool
	Preview                   bool
}

// HistoryResult carries reads, mutations and printing tied to the selected record.
type HistoryResult struct {
	ProjectID, OwnerID, ID int
	Table, Type            string
	Revision               uint64
	Rows, Active           []dto.Document
	Selected               int
	Output, Committed      string
	Err                    error
}

// Invalidate cancels pending HTTP/process work and rejects late results.
func (h History) Invalidate() History {
	if h.cancel != nil {
		h.cancel()
	}
	h.cancel = nil
	h.Revision++
	h.Busy = false
	h.mutating = false
	return h
}

// CancelMutation cancels pending work while retaining its mutation result for reconciliation.
// It reports whether the caller must wait for that result before leaving history.
func (h History) CancelMutation() (History, bool) {
	if !h.Busy || !h.mutating {
		return h, false
	}
	if h.cancel != nil {
		h.cancel()
	}
	return h, true
}

// Current identifies the displayed historical document without inferring active state.
func (h History) Current() (dto.Document, bool) {
	if h.Selected < 0 || h.Selected >= len(h.Rows) {
		return dto.Document{}, false
	}
	return h.Rows[h.Selected], true
}

// Open reads a single owner's selected type, initially showing its newest retained ID.
func (h History) Open(ctx context.Context, api MutationAPI, printer Printer, project, owner int, table, kind string) (History, tea.Cmd) {
	return h.open(ctx, api, printer, project, owner, table, kind, nil)
}

// OpenRecord starts owner history at the explicitly selected retained record.
func (h History) OpenRecord(ctx context.Context, api MutationAPI, printer Printer, project int, record dto.Document) (History, tea.Cmd) {
	return h.open(ctx, api, printer, project, record.RefID, record.RefTable, record.DocType, &record)
}

func (h History) open(ctx context.Context, api MutationAPI, printer Printer, project, owner int, table, kind string, record *dto.Document) (History, tea.Cmd) {
	h = h.Invalidate()
	revision := h.Revision
	h = History{ProjectID: project, OwnerID: owner, Table: table, Type: kind, Revision: revision}
	if err := validScope(project, owner, table); err != nil {
		h.Err = err
		return h, nil
	}
	if strings.TrimSpace(kind) == "" {
		h.Err = errors.New("select a document type")
		return h, nil
	}
	if record != nil {
		h.Rows = []dto.Document{*record}
	}
	return h.begin(ctx, api, printer, true, false, 0)
}

// PreviewBody prints one current item with the same scoped, cancellable viewport as history.
func (h History) PreviewBody(ctx context.Context, printer Printer, project, owner int, table, kind, body string) (History, tea.Cmd) {
	h = h.Invalidate()
	h = History{ProjectID: project, OwnerID: owner, Table: table, Type: kind, Revision: h.Revision, Preview: true}
	if err := validScope(project, owner, table); err != nil {
		h.Err = err
		return h, nil
	}
	work, cancel := context.WithCancel(ctx)
	h.cancel, h.Busy = cancel, true
	result := HistoryResult{ProjectID: project, OwnerID: owner, Table: table, Type: kind, Revision: h.Revision}
	return h, func() tea.Msg {
		if printer == nil {
			result.Err = errors.New("document printer unavailable")
		} else {
			result.Output, result.Err = printer.Print(work, body)
		}
		return result
	}
}

// Refresh reads only, including recovery after a committed activation.
func (h History) Refresh(ctx context.Context, api MutationAPI, printer Printer) (History, tea.Cmd) {
	return h.begin(ctx, api, printer, true, false, h.Selected)
}

// Move prints the adjacent version; Left is newer and Right is older, without wrapping.
func (h History) Move(ctx context.Context, api MutationAPI, printer Printer, delta int) (History, tea.Cmd) {
	if h.Busy || len(h.Rows) == 0 {
		return h, nil
	}
	selected := max(0, min(len(h.Rows)-1, h.Selected+delta))
	if selected == h.Selected {
		return h, nil
	}
	return h.begin(ctx, api, printer, false, false, selected)
}

// Activate selects the same historical ID or undeletes the displayed comment.
func (h History) Activate(ctx context.Context, api MutationAPI, printer Printer) (History, tea.Cmd) {
	if h.Committed != "" && h.Err != nil {
		return h.Refresh(ctx, api, printer)
	}
	d, ok := h.Current()
	if !ok || h.Busy {
		return h, nil
	}
	if d.DocType == "comment" && d.DeletedAt == nil {
		return h, nil
	}
	for _, active := range h.Active {
		if active.ID == d.ID {
			return h, nil
		}
	}
	return h.begin(ctx, api, printer, true, true, h.Selected)
}

func (h History) begin(ctx context.Context, api MutationAPI, printer Printer, read, mutate bool, selected int) (History, tea.Cmd) {
	if h.Busy {
		return h, nil
	}
	h = h.Invalidate()
	h.Selected, h.Offset, h.Output = selected, 0, ""
	h.Busy, h.Err, h.Status = true, nil, "loading history"
	h.mutating = mutate
	work, cancel := context.WithCancel(ctx)
	h.cancel = cancel
	project, owner, table, kind, revision := h.ProjectID, h.OwnerID, h.Table, h.Type, h.Revision
	rows, active, committed := slices.Clone(h.Rows), slices.Clone(h.Active), h.Committed
	id := 0
	if d, ok := h.Current(); ok {
		id = d.ID
	}
	return h, func() tea.Msg {
		defer cancel()
		r := HistoryResult{ProjectID: project, OwnerID: owner, Table: table, Type: kind, Revision: revision, Rows: rows, Active: active, Selected: selected, ID: id, Committed: committed}
		if err := work.Err(); err != nil {
			r.Err = err
			return r
		}
		if mutate {
			if kind == "comment" {
				r.Err = api.UndeleteComment(work, id)
			} else {
				r.Err = api.ActivateDocument(work, id)
			}
			if r.Err != nil {
				return r
			}
			r.Committed = fmt.Sprintf("committed %s document #%d", map[bool]string{true: "undelete", false: "active selection"}[kind == "comment"], id)
		}
		if read {
			var all []dto.Document
			if kind == "comment" {
				all, r.Err = api.ListComments(work, owner, table)
			} else {
				all, r.Err = api.ListDocuments(work, owner, table)
			}
			if r.Err != nil {
				return r
			}
			r.Rows = make([]dto.Document, 0)
			for _, d := range all {
				if d.RefID != owner || d.RefTable != table {
					r.Err = errors.New("history owner differs from request")
					return r
				}
				if d.DocType == kind {
					r.Rows = append(r.Rows, d)
				}
			}
			slices.SortFunc(r.Rows, func(a, b dto.Document) int {
				if a.ID > b.ID {
					return -1
				}
				if a.ID < b.ID {
					return 1
				}
				return 0
			})
			r.Selected = 0
			for i, d := range r.Rows {
				if d.ID == id {
					r.Selected = i
					break
				}
			}
			r.Active, r.Err = api.ActiveDocuments(work, owner, table)
			if r.Err == nil {
				r.Err = ValidateActive(r.Active, owner, table)
			}
			if r.Err != nil {
				return r
			}
		}
		if len(r.Rows) == 0 {
			return r
		}
		if printer == nil {
			r.Err = errors.New("history printer unavailable")
			return r
		}
		r.Output, r.Err = printer.Print(work, r.Rows[r.Selected].Body)
		return r
	}
}

// Apply retains selected history on printing/read failures and rejects obsolete output.
func (h History) Apply(r HistoryResult) (History, bool) {
	if !h.Busy || r.Revision != h.Revision || r.ProjectID != h.ProjectID || r.OwnerID != h.OwnerID || r.Table != h.Table || r.Type != h.Type {
		return h, false
	}
	h = h.Invalidate()
	if r.Rows != nil {
		h.Rows, h.Active, h.Selected = r.Rows, r.Active, r.Selected
	}
	h.Output, h.Err, h.Committed = r.Output, r.Err, r.Committed
	h.Status = "history"
	if h.Preview {
		h.Status = "view " + h.Type
		return h, true
	}
	if d, ok := h.Current(); ok {
		h.Status = fmt.Sprintf("Version #%d | history", d.ID)
	}
	if len(h.Rows) == 0 {
		h.Status = "no retained documents"
	}
	if r.Committed != "" {
		h.Status = r.Committed
	}
	if r.Err != nil {
		h.Status += "; history failed; /retry reads only"
	}
	return h, true
}

// Metadata stays above the scrollable body and uses local timestamps.
func (h History) Metadata() string {
	d, ok := h.Current()
	if !ok {
		return ""
	}
	line := lipgloss.NewStyle().Foreground(styles.AccentCyan).Render("created_at: " + d.CreatedAt.Local().Format("2006-01-02 15:04") + "  updated_at: " + d.UpdatedAt.Local().Format("2006-01-02 15:04"))
	if d.DeletedAt != nil {
		line += "  " + lipgloss.NewStyle().Foreground(styles.AccentRed).Render("deleted_at: "+d.DeletedAt.Local().Format("2006-01-02 15:04"))
	}
	return line
}

// View clips visible cells while retaining only bat's syntax SGR sequences.
func (h History) View(width, height int) string {
	if height <= 0 {
		return ""
	}
	if h.Busy {
		if h.Preview {
			return "Loading view…"
		}
		return "Loading history…"
	}
	if h.Output == "" {
		if h.Preview {
			return "Empty item"
		}
		if len(h.Rows) == 0 {
			return "No retained documents"
		}
		if h.Err != nil {
			return "History output unavailable; /retry"
		}
		return ""
	}
	lines := strings.Split(strings.TrimSuffix(historyText(h.Output), "\n"), "\n")
	start := min(max(0, h.Offset), max(0, len(lines)-height))
	end := min(len(lines), start+height)
	clipped := make([]string, 0, end-start)
	style := ""
	for i, line := range lines[:end] {
		if i >= start {
			// Match Lip Gloss's default four-space tab expansion before clipping.
			line = strings.ReplaceAll(line, "\t", "    ")
			clipped = append(clipped, ansi.Truncate(style+line, max(1, width), "")+ansi.ResetStyle)
		}
		for _, sgr := range historySGR.FindAllString(line, -1) {
			if sgr == "\x1b[m" || sgr == "\x1b[0m" {
				style = ""
			} else if strings.HasPrefix(sgr, "\x1b[0;") {
				style = sgr
			} else {
				style += sgr
			}
		}
	}
	return strings.Join(clipped, "\n")
}

// Scroll advances the body only, without changing metadata or selected version.
func (h History) Scroll(delta, height int) History {
	total := len(strings.Split(strings.TrimSuffix(historyText(h.Output), "\n"), "\n"))
	h.Offset = max(0, min(max(0, total-max(1, height)), h.Offset+delta))
	return h
}

// Help makes the selected record's Space action discoverable.
func (h History) Help() string {
	action := "active selection"
	if h.Type == "comment" {
		action = "undelete comment"
	}
	return "Left newer | Right older | Space " + action + " | Up/Down scroll | PgUp/PgDown page | Esc/Ctrl+C return | /retry"
}

var historySGR = regexp.MustCompile(`\x1b\[[0-9;:]*m`)

// historyText allows text, layout whitespace and complete SGR controls only.
// Decode whole output before splitting lines so multiline control-string payloads
// cannot enter the viewport or affect its scroll bounds.
func historyText(output string) string {
	var safe strings.Builder
	for len(output) > 0 {
		seq, _, n, _ := ansi.DecodeSequence(output, ansi.NormalState, nil)
		if n == 0 {
			output = output[1:]
			continue
		}
		output = output[n:]
		if seq == "\n" || seq == "\t" || (strings.HasPrefix(seq, "\x1b[") && historySGR.FindString(seq) == seq) {
			safe.WriteString(seq)
			continue
		}
		r, _ := utf8.DecodeRuneInString(seq)
		if utf8.ValidString(seq) && !unicode.IsControl(r) {
			safe.WriteString(seq)
		}
	}
	return safe.String()
}
