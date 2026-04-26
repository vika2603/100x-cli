package output

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/mattn/go-isatty"
)

// Table renders rows as an aligned grid when stdout is a terminal, or as
// TSV when stdout is piped. Numeric columns are right-aligned; the
// header is bolded when ANSI is enabled. Headers and cells are rendered
// verbatim — no case or label transformation. Each command owns the
// display labels for its own columns.
func (r *Renderer) Table(headers []string, rows [][]string) error {
	if r.Quiet {
		return nil
	}
	if !r.stdoutIsTTY() {
		return r.tableTSV(headers, rows)
	}
	return r.tableHuman(headers, rows)
}

func (r *Renderer) stdoutIsTTY() bool {
	f, ok := r.Out.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd())
}

func (r *Renderer) tableTSV(headers []string, rows [][]string) error {
	w := csv.NewWriter(r.Out)
	w.Comma = '\t'
	if len(headers) > 0 {
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

func (r *Renderer) tableHuman(headers []string, rows [][]string) error {
	rightAlign := make([]bool, len(headers))
	for i := range rightAlign {
		rightAlign[i] = isNumericColumn(rows, i)
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
			if col >= 0 && col < len(rightAlign) && rightAlign[col] {
				s = s.Align(lipgloss.Right)
			}
			return s
		})
	_, err := fmt.Fprintln(r.Out, t.Render())
	return err
}

func isNumericColumn(rows [][]string, col int) bool {
	saw := false
	for _, row := range rows {
		if col >= len(row) {
			continue
		}
		v := strings.TrimSpace(row[col])
		if v == "" {
			continue
		}
		if _, err := strconv.ParseFloat(v, 64); err != nil {
			return false
		}
		saw = true
	}
	return saw
}
