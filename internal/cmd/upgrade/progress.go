package upgrade

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/imroc/req/v3"
	"github.com/mattn/go-isatty"

	"github.com/vika2603/100x-cli/internal/output"
)

// downloadWithProgress fetches url via req and shows a charmbracelet
// progress bar on stderr while bytes stream in. Falls back to a single
// status line when the renderer isn't attached to a TTY, when --quiet is
// set, or when JSON output is selected.
func downloadWithProgress(ctx context.Context, client *req.Client, url, label string, r *output.Renderer) ([]byte, error) {
	if !canRenderTUI(r) {
		r.Println(label)
		return plainDownload(ctx, client, url)
	}
	return downloadTUI(ctx, client, url, label)
}

func canRenderTUI(r *output.Renderer) bool {
	if r == nil || r.Quiet || r.Format == output.FormatJSON {
		return false
	}
	f, ok := r.Err.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd())
}

func plainDownload(ctx context.Context, client *req.Client, url string) ([]byte, error) {
	var buf bytes.Buffer
	resp, err := client.R().SetContext(ctx).SetOutput(&buf).Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("status %d for %s", resp.StatusCode, url)
	}
	return buf.Bytes(), nil
}

type totalMsg int64
type chunkMsg int64
type finishMsg struct {
	data []byte
	err  error
}

type progModel struct {
	bar      progress.Model
	label    string
	total    int64
	received int64
	final    []byte
	err      error
	finished bool
	cancel   context.CancelFunc
}

func (m progModel) Init() tea.Cmd { return nil }

func (m progModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case totalMsg:
		m.total = int64(msg)
		return m, nil
	case chunkMsg:
		m.received += int64(msg)
		if m.total <= 0 {
			return m, nil
		}
		ratio := float64(m.received) / float64(m.total)
		if ratio > 1 {
			ratio = 1
		}
		return m, m.bar.SetPercent(ratio)
	case finishMsg:
		m.final = msg.data
		m.err = msg.err
		m.finished = true
		return m, tea.Quit
	case progress.FrameMsg:
		pm, cmd := m.bar.Update(msg)
		m.bar = pm.(progress.Model)
		return m, cmd
	case tea.WindowSizeMsg:
		w := msg.Width - 16
		switch {
		case w > 60:
			w = 60
		case w < 20:
			w = 20
		}
		m.bar.Width = w
		return m, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			if m.cancel != nil {
				m.cancel()
			}
			m.err = errors.New("aborted")
			m.finished = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m progModel) View() string {
	if m.finished {
		return ""
	}
	dim := lipgloss.NewStyle().Faint(true)
	stats := dim.Render(progressStats(m.received, m.total))
	return fmt.Sprintf("%s\n  %s  %s\n", m.label, m.bar.View(), stats)
}

func progressStats(received, total int64) string {
	var rec uint64
	if received > 0 {
		rec = uint64(received)
	}
	if total > 0 {
		return fmt.Sprintf("%s / %s", humanize.Bytes(rec), humanize.Bytes(uint64(total)))
	}
	return humanize.Bytes(rec)
}

func downloadTUI(parent context.Context, client *req.Client, url, label string) ([]byte, error) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	prog := tea.NewProgram(
		progModel{
			bar:    progress.New(progress.WithDefaultGradient(), progress.WithWidth(40)),
			label:  label,
			cancel: cancel,
		},
		tea.WithOutput(os.Stderr),
		tea.WithContext(ctx),
	)

	go runRequest(ctx, client, url, prog)

	final, err := prog.Run()
	if err != nil {
		// If the program was killed via context cancellation, surface the
		// abort directly rather than the generic ErrProgramKilled.
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil, errors.New("aborted")
		}
		return nil, err
	}
	out := final.(progModel)
	if out.err != nil {
		return nil, out.err
	}
	return out.final, nil
}

func runRequest(ctx context.Context, client *req.Client, url string, prog *tea.Program) {
	var (
		buf       bytes.Buffer
		mu        sync.Mutex
		announced bool
		lastSize  int64
	)
	resp, err := client.R().
		SetContext(ctx).
		SetOutput(&buf).
		SetDownloadCallback(func(info req.DownloadInfo) {
			mu.Lock()
			defer mu.Unlock()
			if !announced && info.Response != nil && info.Response.ContentLength > 0 {
				announced = true
				prog.Send(totalMsg(info.Response.ContentLength))
			}
			delta := info.DownloadedSize - lastSize
			lastSize = info.DownloadedSize
			if delta > 0 {
				prog.Send(chunkMsg(delta))
			}
		}).
		Get(url)
	if err != nil {
		prog.Send(finishMsg{err: err})
		return
	}
	if resp.StatusCode/100 != 2 {
		prog.Send(finishMsg{err: fmt.Errorf("status %d for %s", resp.StatusCode, url)})
		return
	}
	prog.Send(finishMsg{data: buf.Bytes()})
}
