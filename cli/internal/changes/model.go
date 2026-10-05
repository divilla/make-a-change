package changes

import (
	"cli/internal/dto"
	"cli/internal/ui"
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Filters stores active change list filter selections.
type Filters struct {
	Phase dto.Option
	Epic  dto.Option
	Type  dto.Option
	Find  string
}

// PhaseColors maps backend phase slugs to optional Lip Gloss color values.
type PhaseColors map[string]string

// Model stores changes list and detail state.
type Model struct {
	ProjectID, EntityID int
	Generation          uint64
	Operation           Operation
	Busy, DetailLoaded  bool
	Status, Outcome     string
	Err                 error
	Draft               Input
	refreshOp           Operation
	refreshID           int
	cancel              context.CancelFunc

	Rows           []dto.ChangeView
	Selected       int
	Offset         int
	Detail         dto.ChangeView
	DetailSelected int
	DetailOffset   int
	Loading        bool
	Inactive       bool
}

// DetailRow is one row in the Change details table.
type DetailRow struct {
	Label        string
	Text         string
	Selectable   bool
	DividerAfter bool
	TestCaseID   string
	TestCaseText string
	TestCaseDone bool
	DocumentType string
	DocumentID   int
	Comment      bool
	NoWrap       bool
	Timestamp    string
}

// ParsedSpec stores metadata extracted from spec markdown.
type ParsedSpec struct {
	Title              string
	Spec               string
	ChangeTypes        []string
	ChangeTypesPresent bool
}

// ParsedBrief stores the title and full brief text extracted from brief markdown.
type ParsedBrief struct {
	Title              string
	Brief              string
	ChangeTypes        []string
	ChangeTypesPresent bool
}

var (
	invalidArtifactTypeCharacters = regexp.MustCompile(`[^A-Za-z\-_]`)
	findWordSeparators            = regexp.MustCompile(`[^a-z0-9_-]+`)
)

// StartLoading returns a changes model in loading state.
func StartLoading() Model {
	return Model{Loading: true}
}

// WithRows returns a changes model populated with loaded rows.
func (m Model) WithRows(rows []dto.ChangeView) Model {
	m.Rows = rows
	m.Selected = 0
	m.Offset = 0
	m.Loading = false
	return m
}

// WithError returns a changes model reset after load failure.
func (m Model) WithError() Model {
	m.Rows = nil
	m.Selected = 0
	m.Offset = 0
	m.Loading = false
	return m
}

// MoveSelection moves the selected change within list bounds.
func (m Model) MoveSelection(delta int, filters Filters, pageSize int) Model {
	m = m.ClampSelection(filters, pageSize)
	visible := FilteredRows(m.Rows, filters)
	if len(visible) == 0 {
		return m
	}
	next := m.Selected + delta
	if next < 0 {
		next = 0
	}
	if next >= len(visible) {
		next = len(visible) - 1
	}
	m.Selected = next
	m.Offset = clampOffset(m.Offset, m.Selected, len(visible), pageSize)
	return m
}

// ClampSelection keeps the selected visible row and scroll offset in bounds.
func (m Model) ClampSelection(filters Filters, pageSize int) Model {
	visible := FilteredRows(m.Rows, filters)
	if len(visible) == 0 {
		m.Selected = 0
		m.Offset = 0
		return m
	}
	if m.Selected < 0 {
		m.Selected = 0
	}
	if m.Selected >= len(visible) {
		m.Selected = len(visible) - 1
	}
	m.Offset = clampOffset(m.Offset, m.Selected, len(visible), pageSize)
	return m
}

// SelectDetail selects the current visible change.
func (m Model) SelectDetail(filters Filters) (Model, dto.ChangeView, bool) {
	visible := FilteredRows(m.Rows, filters)
	if m.Loading || len(visible) == 0 {
		return m, dto.ChangeView{}, false
	}
	m = m.ClampSelection(filters, 1)
	m.Offset = clampOffset(m.Offset, m.Selected, len(visible), 1)
	selected := visible[m.Selected]
	m = m.WithDetail(selected)
	return m, selected, true
}

// WithDetail stores the selected Change and resets detail-table selection.
func (m Model) WithDetail(change dto.ChangeView) Model {
	m.Detail = change
	m.DetailSelected = firstSelectableDetailSelection(change)
	m.DetailOffset = 0
	return m
}

// MoveDetailSelection moves within editable detail rows and keeps the row visible.
func (m Model) MoveDetailSelection(delta int, pageSize int, width int) Model {
	rows := DetailRows(m.Detail)
	if len(rows) == 0 {
		m.DetailSelected = 0
		m.DetailOffset = 0
		return m
	}
	m = m.ClampDetailSelection(pageSize, width)
	next := nextSelectableDetailSelection(m.Detail, rows, m.DetailSelected, delta)
	rows, prefix := detailViewportRows(m.Detail, pageSize, width)
	m.DetailSelected = next
	if m.DetailSelected+prefix < 0 {
		_, textWidth := DetailColumnWidths(m.Detail, width)
		m.DetailOffset = clampLineOffset(m.DetailOffset, detailLineCount(rows, textWidth), detailScrollPageSize(m.Detail, pageSize, width))
		return m
	}
	_, textWidth := DetailColumnWidths(m.Detail, width)
	rowStart := detailRowLineStart(rows, m.DetailSelected+prefix, textWidth)
	rowEnd := rowStart + detailRowLineCount(rows[m.DetailSelected+prefix], textWidth)
	m.DetailOffset = detailOffsetKeepingRowVisible(m.DetailOffset, rowStart, rowEnd, detailLineCount(rows, textWidth), detailScrollPageSize(m.Detail, pageSize, width))
	return m
}

// ClampDetailSelection keeps the selected detail row and scroll offset in bounds.
func (m Model) ClampDetailSelection(pageSize int, width int) Model {
	rows := DetailRows(m.Detail)
	if len(rows) == 0 {
		m.DetailSelected = 0
		m.DetailOffset = 0
		return m
	}
	if !validDetailSelection(m.Detail, rows, m.DetailSelected) {
		m.DetailSelected = firstSelectableDetailSelection(m.Detail)
	}
	rows, _ = detailViewportRows(m.Detail, pageSize, width)
	_, textWidth := DetailColumnWidths(m.Detail, width)
	m.DetailOffset = clampLineOffset(m.DetailOffset, detailLineCount(rows, textWidth), detailScrollPageSize(m.Detail, pageSize, width))
	return m
}

// ScrollDetailViewport moves the detail table viewport by rendered lines.
func (m Model) ScrollDetailViewport(delta int, pageSize int, width int) Model {
	rows := DetailRows(m.Detail)
	if len(rows) == 0 {
		m.DetailSelected = 0
		m.DetailOffset = 0
		return m
	}
	rows, prefix := detailViewportRows(m.Detail, pageSize, width)
	_, textWidth := DetailColumnWidths(m.Detail, width)
	scrollPageSize := detailScrollPageSize(m.Detail, pageSize, width)
	if abs(delta) >= pageSize {
		if delta < 0 {
			delta = -scrollPageSize
		} else {
			delta = scrollPageSize
		}
	}
	m.DetailOffset = clampLineOffset(m.DetailOffset+delta, detailLineCount(rows, textWidth), scrollPageSize)
	m.DetailSelected = selectableDetailRowAtOffset(rows, m.DetailOffset, textWidth) - prefix
	return m
}

// SelectDetailRow returns the currently selected editable detail row.
func (m Model) SelectDetailRow(pageSize int, width int) (Model, DetailRow, bool) {
	m = m.ClampDetailSelection(pageSize, width)
	rows := DetailRows(m.Detail)
	row, ok := detailRowForSelection(m.Detail, rows, m.DetailSelected)
	if !ok {
		return m, DetailRow{}, false
	}
	return m, row, true
}

// DetailRowAtSelection identifies a row without changing the viewport.
func DetailRowAtSelection(change dto.ChangeView, selection int) (DetailRow, bool) {
	return detailRowForSelection(change, DetailRows(change), selection)
}

// DetailRows returns Change details as label/text table rows.
func DetailRows(change dto.ChangeView) []DetailRow {
	if change.ID == "" && change.Title == "" {
		return nil
	}
	rows := []DetailRow{
		{Label: "Epic", Text: displayNullable(change.EpicName), Selectable: true},
		{Label: "Phase", Text: change.ChangePhase, Selectable: true},
		{Label: "Types", Text: strings.Join(change.ChangeTypes, "|"), Selectable: true},
		{Label: "Active", Text: testCaseDoneIcon(change.Active), Selectable: true},
		{Label: "Timestamps", Text: "Created: " + formatListTimestamp(change.Created) + " -=- Modified: " + formatListTimestamp(change.Modified), DividerAfter: true},
		{Label: "Title", Text: change.Title, Selectable: true, DividerAfter: true},
	}
	for _, kind := range change.DocumentTypes {
		if kind == "comment" {
			continue
		}
		row := DetailRow{Text: "[ ] " + kind, Selectable: true, DocumentType: kind, NoWrap: true}
		for _, d := range change.Documents {
			if d.DocType == kind {
				row.DocumentID = d.ID
				row.Text = "[✓] " + kind
				if !d.UpdatedAt.IsZero() {
					row.Timestamp = d.UpdatedAt.Local().Format("2006-01-02 15:04")
				}
			}
		}
		if len(rows) == 6 {
			row.Label = "Docs"
		}
		rows = append(rows, row)
	}
	if len(rows) == 6 {
		rows = append(rows, DetailRow{Label: "Docs", Text: "-"})
	}
	rows[len(rows)-1].DividerAfter = true
	for i, testCase := range change.TestCases {
		check := "[ ]"
		if testCase.Done {
			check = "[✓]"
		}
		row := DetailRow{
			Text:         check + " " + ui.SafeText(normalizeNewlines(testCase.Scenario)) + fmt.Sprintf(" (#%d)", testCase.ID),
			Selectable:   true,
			TestCaseID:   strconv.Itoa(testCase.ID),
			TestCaseText: testCase.Scenario,
			TestCaseDone: testCase.Done,
		}
		if i == 0 {
			row.Label = "Testcases"
		}
		rows = append(rows, row)
	}
	if len(change.TestCases) == 0 {
		rows = append(rows, DetailRow{Label: "Testcases", Text: "-"})
	}
	rows[len(rows)-1].DividerAfter = true
	rows = append(rows, DetailRow{Label: "Completed", Text: fmt.Sprintf("---=== %d/%d - %d%% ===---", change.Done, change.Total, change.Completed), DividerAfter: true})
	commentStart := len(rows)
	for _, d := range change.Comments {
		if d.DeletedAt != nil {
			continue
		}
		row := DetailRow{Text: "- " + strings.ReplaceAll(ui.SafeText(normalizeNewlines(d.Body)), "\n", " "), Selectable: true, DocumentType: "comment", DocumentID: d.ID, Comment: true, NoWrap: true}
		if !d.UpdatedAt.IsZero() {
			row.Timestamp = d.UpdatedAt.Local().Format("2006-01-02 15:04")
			row.Text = row.Timestamp + " " + row.Text
		}
		if len(rows) == commentStart {
			row.Label = "Comments"
		}
		rows = append(rows, row)
	}
	if len(rows) == commentStart {
		rows = append(rows, DetailRow{Label: "Comments", Text: "-", Selectable: true, Comment: true, DocumentType: "comment"})
	}
	rows[len(rows)-1].DividerAfter = true
	rows = append(rows,
		DetailRow{Label: "After Change", Text: displayNullable(change.AfterChangeName), Selectable: true},
		DetailRow{Label: "Ref UUID", Text: displayNullable(change.RefUUID), Selectable: true},
		DetailRow{Label: "Slug", Text: displayNullable(change.RefSlug), Selectable: true},
		DetailRow{Label: "PR URL", Text: displayNullable(change.PRUrl), Selectable: true},
	)
	return rows
}

func fixedDetailRows(change dto.ChangeView) []DetailRow {
	return []DetailRow{
		{Label: "ID", Text: change.ID, Selectable: true},
	}
}

// DetailCopyValue returns the value copied for a selected detail row.
func DetailCopyValue(row DetailRow) string {
	return row.Text
}

func firstSelectableDetailSelection(change dto.ChangeView) int {
	fixedRows := fixedDetailRows(change)
	for i, row := range fixedRows {
		if row.Selectable {
			return i - len(fixedRows)
		}
	}
	return firstSelectableDetailRow(DetailRows(change))
}

func validDetailSelection(change dto.ChangeView, rows []DetailRow, selected int) bool {
	_, ok := detailRowForSelection(change, rows, selected)
	return ok
}

func detailRowForSelection(change dto.ChangeView, rows []DetailRow, selected int) (DetailRow, bool) {
	if selected < 0 {
		fixedRows := fixedDetailRows(change)
		index := selected + len(fixedRows)
		if index < 0 || index >= len(fixedRows) || !fixedRows[index].Selectable {
			return DetailRow{}, false
		}
		return fixedRows[index], true
	}
	if selected >= len(rows) || !rows[selected].Selectable {
		return DetailRow{}, false
	}
	return rows[selected], true
}

func selectableDetailSelections(change dto.ChangeView, rows []DetailRow) []int {
	fixedRows := fixedDetailRows(change)
	selections := make([]int, 0, len(fixedRows)+len(rows))
	for i, row := range fixedRows {
		if row.Selectable {
			selections = append(selections, i-len(fixedRows))
		}
	}
	for i, row := range rows {
		if row.Selectable {
			selections = append(selections, i)
		}
	}
	return selections
}

func nextSelectableDetailSelection(change dto.ChangeView, rows []DetailRow, selected int, delta int) int {
	selections := selectableDetailSelections(change, rows)
	if len(selections) == 0 || delta == 0 {
		return selected
	}
	position := -1
	for i, value := range selections {
		if value == selected {
			position = i
			break
		}
	}
	if position < 0 {
		return selections[0]
	}
	next := position + delta
	if next < 0 {
		next = 0
	}
	if next >= len(selections) {
		next = len(selections) - 1
	}
	return selections[next]
}

// detailViewportRows lets identity rows scroll when pinning them would hide the body.
// The prefix translates viewport indexes to the stable detail selection indexes.
func detailViewportRows(change dto.ChangeView, pageSize, width int) ([]DetailRow, int) {
	rows := DetailRows(change)
	if detailFixedLineCount(change, width) >= max(1, pageSize) {
		fixed := fixedDetailRows(change)
		return append(fixed, rows...), len(fixed)
	}
	return rows, 0
}

func detailFixedLineCount(change dto.ChangeView, width int) int {
	_, textWidth := DetailColumnWidths(change, width)
	lines := 0
	for _, row := range fixedDetailRows(change) {
		lines += detailRowLineCount(row, textWidth)
	}
	return lines
}

func detailScrollPageSize(change dto.ChangeView, pageSize int, width int) int {
	pageSize = max(1, pageSize)
	fixed := detailFixedLineCount(change, width)
	if fixed >= pageSize {
		return pageSize
	}
	return pageSize - fixed
}

func testCaseDoneIcon(value bool) string {
	if value {
		return "\u2705"
	}
	return "\u274c"
}

// DetailColumnWidths returns label and text widths for the rendered details table.
func DetailColumnWidths(change dto.ChangeView, width int) (int, int) {
	contentWidth := width
	if width <= 4 {
		contentWidth = 20
	}
	if contentWidth < 20 {
		contentWidth = 20
	}
	labelWidth := detailLabelWidth(DetailRows(change))
	textWidth := contentWidth - labelWidth - 3
	if textWidth < 10 {
		textWidth = 10
		labelWidth = max(1, contentWidth-textWidth-3)
	}
	return labelWidth, textWidth
}

func firstSelectableDetailRow(rows []DetailRow) int {
	for i, row := range rows {
		if row.Selectable {
			return i
		}
	}
	return 0
}

func detailLineCount(rows []DetailRow, textWidth int) int {
	total := 0
	for _, row := range rows {
		total += detailRowLineCount(row, textWidth)
		if detailDividerAfter(row) {
			total++
		}
	}
	return total
}

func detailRowLineStart(rows []DetailRow, rowIndex int, textWidth int) int {
	start := 0
	for i, row := range rows {
		if i == rowIndex {
			return start
		}
		start += detailRowLineCount(row, textWidth)
		if detailDividerAfter(row) {
			start++
		}
	}
	return start
}

func detailRowLineCount(row DetailRow, textWidth int) int {
	return len(detailRowTextLines(row, textWidth))
}

func selectableDetailRowAtOffset(rows []DetailRow, offset int, textWidth int) int {
	line := 0
	for i, row := range rows {
		count := detailRowLineCount(row, textWidth)
		if row.Selectable && line+count > offset {
			return i
		}
		line += count
		if detailDividerAfter(row) {
			line++
		}
	}
	for i := len(rows) - 1; i >= 0; i-- {
		if rows[i].Selectable {
			return i
		}
	}
	return 0
}

func detailRowTextLines(row DetailRow, textWidth int) []string {
	if row.NoWrap {
		value := row.Text
		if row.DocumentType != "" && !row.Comment {
			value = padRightDisplay(row.Timestamp, len("2006-01-02 15:04")) + " - " + value
		}
		return strings.Split(value, "\n")
	}
	value := strings.TrimSpace(row.Text)
	if value == "" {
		value = "-"
	}
	parts := strings.Split(normalizeNewlines(value), "\n")
	textLines := make([]string, 0, len(parts))
	for _, part := range parts {
		wrapped := wrapWords(part, textWidth)
		if len(wrapped) == 0 {
			wrapped = []string{""}
		}
		textLines = append(textLines, wrapped...)
	}
	return textLines
}

func detailDividerAfter(row DetailRow) bool {
	return row.DividerAfter
}

func clampOffset(offset, selected, total, pageSize int) int {
	if total <= 0 {
		return 0
	}
	if pageSize < 1 {
		pageSize = 1
	}
	if selected < 0 {
		selected = 0
	}
	if selected >= total {
		selected = total - 1
	}
	if offset > selected {
		offset = selected
	}
	if selected >= offset+pageSize {
		offset = selected - pageSize + 1
	}
	maxOffset := total - pageSize
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	if offset < 0 {
		return 0
	}
	return offset
}

func clampLineOffset(offset, total, pageSize int) int {
	if total <= 0 {
		return 0
	}
	if pageSize < 1 {
		pageSize = 1
	}
	maxOffset := total - pageSize
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset > maxOffset {
		return maxOffset
	}
	if offset < 0 {
		return 0
	}
	return offset
}

func detailOffsetKeepingRowVisible(offset, rowStart, rowEnd, total, pageSize int) int {
	offset = clampLineOffset(offset, total, pageSize)
	if pageSize < 1 {
		pageSize = 1
	}
	if rowEnd <= rowStart {
		rowEnd = rowStart + 1
	}
	if rowStart < offset {
		return clampLineOffset(rowStart, total, pageSize)
	}
	if rowEnd > offset+pageSize {
		return clampLineOffset(rowEnd-pageSize, total, pageSize)
	}
	return offset
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

// FilteredRows returns changes matching active filters.
func FilteredRows(rows []dto.ChangeView, filters Filters) []dto.ChangeView {
	filtered := make([]dto.ChangeView, 0, len(rows))
	findWords := normalizedFindWords(filters.Find)
	for _, change := range rows {
		if filters.Phase.ID != "" && change.ChangePhase != filters.Phase.ID && change.ChangePhase != filters.Phase.Label {
			continue
		}
		if filters.Type.ID != "" && !hasChangeType(change, filters.Type.ID, filters.Type.Label) {
			continue
		}
		if filters.Epic.ID != "" && change.EpicID != filters.Epic.ID && change.EpicName != filters.Epic.Label {
			continue
		}
		if len(findWords) > 0 && !matchesFind(change, findWords) {
			continue
		}
		filtered = append(filtered, change)
	}
	return filtered
}

// ParseBriefStructure extracts the Change title and brief text.
func ParseBriefStructure(brief string) (ParsedBrief, error) {
	normalized := strings.ReplaceAll(strings.ReplaceAll(brief, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(normalized, "\n")
	firstIndex := firstNonBlankLine(lines, 0)
	if firstIndex < 0 || !strings.HasPrefix(strings.TrimSpace(lines[firstIndex]), "# ") || strings.HasPrefix(strings.TrimSpace(lines[firstIndex]), "## ") {
		return ParsedBrief{}, fmt.Errorf("brief title is required")
	}
	title := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[firstIndex]), "# "))
	if title == "" {
		return ParsedBrief{}, fmt.Errorf("brief title is required")
	}
	types, typesPresent := ParseArtifactTypes(normalized)
	return ParsedBrief{
		Title:              title,
		Brief:              brief,
		ChangeTypes:        types,
		ChangeTypesPresent: typesPresent,
	}, nil
}

// ParseSpecStructure extracts the title, optional Types metadata, and body.
func ParseSpecStructure(spec string) (ParsedSpec, error) {
	normalized := strings.ReplaceAll(strings.ReplaceAll(spec, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], "# ") || strings.HasPrefix(lines[0], "## ") {
		return ParsedSpec{}, fmt.Errorf("spec title is required")
	}
	title := strings.TrimSpace(strings.TrimPrefix(lines[0], "# "))
	if title == "" {
		return ParsedSpec{}, fmt.Errorf("spec title is required")
	}
	if len(lines) < 2 || strings.TrimSpace(lines[1]) != "" {
		return ParsedSpec{}, fmt.Errorf("spec title must be followed by one blank line")
	}
	types, typesPresent := ParseArtifactTypes(normalized)
	bodyLines := lines[2:]
	firstBodyLine := firstNonBlankLine(bodyLines, 0)
	if firstBodyLine >= 0 && isArtifactTypesLine(bodyLines[firstBodyLine]) {
		bodyLines = bodyLines[firstBodyLine+1:]
	}
	if strings.TrimSpace(strings.Join(bodyLines, "\n")) == "" {
		return ParsedSpec{}, fmt.Errorf("spec body is required")
	}

	return ParsedSpec{
		Title:              title,
		Spec:               normalized,
		ChangeTypes:        types,
		ChangeTypesPresent: typesPresent,
	}, nil
}

// ParseArtifactTypes returns optional Types metadata without catalog validation.
func ParseArtifactTypes(artifact string) ([]string, bool) {
	normalized := strings.ReplaceAll(strings.ReplaceAll(artifact, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(normalized, "\n")
	firstIndex := firstNonBlankLine(lines, 0)
	if firstIndex < 0 {
		return nil, false
	}
	metadataIndex := firstIndex
	firstLine := strings.TrimSpace(lines[firstIndex])
	if strings.HasPrefix(firstLine, "# ") && !strings.HasPrefix(firstLine, "## ") {
		metadataIndex = firstNonBlankLine(lines, firstIndex+1)
	}
	if metadataIndex < 0 || !isArtifactTypesLine(lines[metadataIndex]) {
		return nil, false
	}
	value := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[metadataIndex]), "Types:"))
	fields := strings.Fields(strings.ReplaceAll(value, "|", " "))
	values := make([]string, 0, len(fields))
	for _, field := range fields {
		cleaned := invalidArtifactTypeCharacters.ReplaceAllString(field, "")
		if cleaned != "" {
			values = append(values, cleaned)
		}
	}
	return values, true
}

func isArtifactTypesLine(line string) bool {
	line = strings.TrimSpace(line)
	return strings.HasPrefix(line, "Types:")
}

// SpecMarkdown returns editable spec markdown for a change.
func SpecMarkdown(change dto.ChangeView) string {
	spec := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(change.Spec, "\r\n", "\n"), "\r", "\n"))
	if _, err := ParseSpecStructure(spec); err == nil {
		return change.Spec
	}

	title := strings.TrimSpace(change.Title)
	body := spec
	lines := strings.Split(spec, "\n")
	if len(lines) > 0 && strings.HasPrefix(lines[0], "# ") && !strings.HasPrefix(lines[0], "## ") {
		title = strings.TrimSpace(strings.TrimPrefix(lines[0], "# "))
		bodyLines := lines[1:]
		for len(bodyLines) > 0 && strings.TrimSpace(bodyLines[0]) == "" {
			bodyLines = bodyLines[1:]
		}
		if len(bodyLines) > 0 && (bodyLines[0] == "Types:" || strings.HasPrefix(bodyLines[0], "Types: ")) {
			bodyLines = bodyLines[1:]
			for len(bodyLines) > 0 && strings.TrimSpace(bodyLines[0]) == "" {
				bodyLines = bodyLines[1:]
			}
		}
		body = strings.TrimSpace(strings.Join(bodyLines, "\n"))
	}

	result := "# " + title + "\n\n"
	typeLine := "Types:"
	if len(change.ChangeTypes) > 0 {
		typeLine += " " + strings.Join(change.ChangeTypes, "|")
	}
	result += typeLine + "\n\n"
	if body != "" {
		result += body
	}
	return strings.TrimRight(result, "\n")
}

func firstNonBlankLine(lines []string, start int) int {
	for i := start; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			return i
		}
	}
	return -1
}

func hasChangeType(change dto.ChangeView, values ...string) bool {
	for _, changeType := range change.ChangeTypes {
		for _, value := range values {
			if value != "" && changeType == value {
				return true
			}
		}
	}
	return false
}

func normalizedFindWords(value string) []string {
	return strings.Fields(findWordSeparators.ReplaceAllString(strings.ToLower(value), " "))
}

func matchesFind(change dto.ChangeView, queryWords []string) bool {
	rowWords := normalizedFindWords(strings.Join([]string{
		displayRef(change),
		change.ChangePhase,
		strings.Join(change.ChangeTypes, " "),
		epicLabel(change),
		change.Title,
	}, " "))
	for _, queryWord := range queryWords {
		found := false
		for _, rowWord := range rowWords {
			if strings.HasPrefix(rowWord, queryWord) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// SortedTypeOptions returns deterministic type options for tests and rendering.
func SortedTypeOptions(options []dto.Option) []dto.Option {
	sorted := append([]dto.Option(nil), options...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Label < sorted[j].Label
	})
	return sorted
}
