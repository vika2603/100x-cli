package output

import (
	"encoding/csv"
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/mattn/go-isatty"
)

// Align controls the horizontal alignment of a column's cells.
type Align int

// Align values.
const (
	// AlignLeft left-aligns column cells. Use for identifiers, names, and free-form text.
	AlignLeft Align = iota
	// AlignRight right-aligns column cells. Use for numeric values so decimal points line up.
	AlignRight
)

// Column declares one column of a table: its header label and its
// alignment. Alignment is a column property, not derived from the cell
// contents — this keeps formatting choices in the caller (colors, units,
// `-` placeholders, percent signs) from accidentally changing layout.
type Column struct {
	Header string
	Align  Align
}

// LCol is a left-aligned column. Use for identifiers, names, statuses,
// timestamps, and any free-form text.
func LCol(header string) Column { return Column{Header: header, Align: AlignLeft} }

// RCol is a right-aligned column. Use for quantities, prices, sizes,
// percentages, and other numeric values where decimal points should line up.
func RCol(header string) Column { return Column{Header: header, Align: AlignRight} }

// Table renders rows as an aligned grid when stdout is a terminal, or as
// TSV when stdout is piped. Per-column alignment is declared by the
// caller via Column. Headers and cells are rendered verbatim — no case
// or label transformation.
func (r *Renderer) Table(cols []Column, rows [][]string) error {
	if r.Quiet {
		return nil
	}
	if !r.stdoutIsTTY() {
		return r.tableTSV(cols, rows)
	}
	return r.tableHuman(cols, rows)
}

func (r *Renderer) stdoutIsTTY() bool {
	f, ok := r.Out.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd())
}

func (r *Renderer) tableTSV(cols []Column, rows [][]string) error {
	w := csv.NewWriter(r.Out)
	w.Comma = '\t'
	if len(cols) > 0 {
		headers := make([]string, len(cols))
		for i, c := range cols {
			headers[i] = c.Header
		}
		if err := w.Write(headers); err != nil {
			return err
		}
	}
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func (r *Renderer) tableHuman(cols []Column, rows [][]string) error {
	headers := make([]string, len(cols))
	for i, c := range cols {
		headers[i] = c.Header
	}
	headerStyle := lipgloss.NewStyle().Padding(0, 2, 0, 0).Bold(r.ColorOnStdout())
	cellStyle := lipgloss.NewStyle().Padding(0, 2, 0, 0)
	t := table.New().
		Border(lipgloss.HiddenBorder()).
		BorderTop(false).BorderBottom(false).
		BorderLeft(false).BorderRight(false).
		BorderColumn(false).BorderRow(false).BorderHeader(false).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			s := cellStyle
			if row == table.HeaderRow {
				s = headerStyle
			}
			if col >= 0 && col < len(cols) && cols[col].Align == AlignRight {
				s = s.Align(lipgloss.Right)
			}
			return s
		})
	_, err := fmt.Fprintln(r.Out, t.Render())
	return err
}
