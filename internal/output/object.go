package output

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// KV is a single key/value pair in an Object render.
type KV struct {
	Key   string
	Value string
}

// Object renders a single record as a two-column key/value list.
//
// In tty mode keys are bolded and the value column aligned; in piped
// mode the same data is emitted as tab-separated `key\tvalue` lines so
// callers can `cut -f1` for keys or `awk -F'\t'` for values. We use
// lipgloss.JoinVertical / JoinHorizontal rather than lipgloss/table —
// the table helper drops the last row when no headers are set (8+ rows
// reproduce; reported upstream).
func (r *Renderer) Object(pairs []KV) error {
	if r.Quiet || len(pairs) == 0 {
		return nil
	}
	if !r.stdoutIsTTY() {
		for _, p := range pairs {
			if _, err := fmt.Fprintf(r.Out, "%s\t%s\n", p.Key, p.Value); err != nil {
				return err
			}
		}
		return nil
	}
	keyStyle := lipgloss.NewStyle().Padding(0, 2, 0, 0).Bold(r.ColorOnStdout())
	keys := make([]string, len(pairs))
	vals := make([]string, len(pairs))
	for i, p := range pairs {
		keys[i] = keyStyle.Render(p.Key)
		vals[i] = p.Value
	}
	keyCol := lipgloss.JoinVertical(lipgloss.Left, keys...)
	valCol := lipgloss.JoinVertical(lipgloss.Left, vals...)
	out := lipgloss.JoinHorizontal(lipgloss.Top, keyCol, valCol)
	_, err := fmt.Fprintln(r.Out, out)
	return err
}
