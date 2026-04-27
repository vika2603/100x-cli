package root

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

// helpTemplate renders root and subcommand --help. Sections are emitted
// in a fixed order: description, usage, aliases, examples, commands,
// flags, global flags. `$` is the root cobra.Command passed to Execute
// and stays stable across ranges, so it's used to compute padding that
// must align across nested loops.
const helpTemplate = `
{{- with or .Long .Short -}}
{{. | trimTrailingWhitespaces}}
{{- end}}

{{section "Usage"}}
  {{.UseLine}}{{commandSuffix .}}

{{- if .Aliases}}

{{section "Aliases"}}
  {{join .Aliases ", "}}
{{- end}}

{{- if .Example}}

{{section "Examples"}}
{{formatExamples .Example}}
{{- end}}

{{- if .HasAvailableSubCommands}}
{{- if groupedCommands .}}
{{- range groupedCommands .}}
{{- if .Commands}}

{{section .Title}}
{{- range .Commands}}
  {{commandName (rpad (commandLabel .) (commandNamePadding $))}} {{.Short}}
{{- end}}
{{- end}}
{{- end}}
{{- if ungroupedCommands .}}

{{section "Additional Commands"}}
{{- range ungroupedCommands .}}
  {{commandName (rpad (commandLabel .) (commandNamePadding $))}} {{.Short}}
{{- end}}
{{- end}}
{{- else}}

{{section "Commands"}}
{{- range ungroupedCommands .}}
  {{commandName (rpad (commandLabel .) (commandNamePadding $))}} {{.Short}}
{{- end}}
{{- end}}
{{- end}}

{{- if .HasAvailableLocalFlags}}

{{section "Flags"}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
{{- end}}

{{- if .HasAvailableInheritedFlags}}

{{section "Global Flags"}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}
{{- end}}
`

type commandGroupView struct {
	Title    string
	Commands []*cobra.Command
}

func configureHelp(cmd *cobra.Command) {
	applyHelp(cmd)
}

func applyHelp(cmd *cobra.Command) {
	cmd.Flags().SortFlags = false
	cmd.PersistentFlags().SortFlags = false
	if cmd.HasSubCommands() && !cmd.Runnable() {
		cmd.Args = cobra.NoArgs
		cmd.RunE = func(c *cobra.Command, args []string) error {
			if err := cobra.NoArgs(c, args); err != nil {
				return err
			}
			return c.Help()
		}
	}
	cmd.SetHelpFunc(func(c *cobra.Command, _ []string) {
		if err := renderHelp(c.OutOrStdout(), c); err != nil {
			_, _ = fmt.Fprintln(c.ErrOrStderr(), err)
		}
	})
	cmd.SetUsageFunc(func(c *cobra.Command) error {
		return renderHelp(c.OutOrStdout(), c)
	})
	cmd.InitDefaultHelpFlag()
	if hf := cmd.Flags().Lookup("help"); hf != nil {
		hf.Usage = "Show help for command"
	}
	for _, sub := range cmd.Commands() {
		applyHelp(sub)
	}
}

func renderHelp(w io.Writer, cmd *cobra.Command) error {
	styler := newHelpStyler(cmd, w)
	tmpl, err := template.New("help").Funcs(template.FuncMap{
		"join":                    strings.Join,
		"groupedCommands":         groupedCommands,
		"ungroupedCommands":       ungroupedCommands,
		"commandLabel":            commandLabel,
		"commandNamePadding":      commandNamePadding,
		"rpad":                    rpad,
		"section":                 styler.section,
		"commandName":             styler.commandName,
		"commandSuffix":           commandSuffix,
		"formatExamples":          styler.formatExamples,
		"trimTrailingWhitespaces": trimTrailingWhitespaces,
	}).Parse(helpTemplate)
	if err != nil {
		return err
	}
	if err := tmpl.Execute(w, cmd); err != nil {
		return err
	}
	_, err = fmt.Fprintln(w)
	return err
}

func commandSuffix(cmd *cobra.Command) string {
	if !cmd.HasAvailableSubCommands() {
		return ""
	}
	if cmd.Annotations["100x-default-command"] == "list" {
		return " [command]"
	}
	return " <command>"
}

func commandLabel(cmd *cobra.Command) string {
	if len(cmd.Aliases) == 0 {
		return cmd.Name()
	}
	return fmt.Sprintf("%s (%s)", cmd.Name(), strings.Join(cmd.Aliases, ", "))
}

func commandNamePadding(cmd *cobra.Command) int {
	maxWidth := 0
	for _, sub := range cmd.Commands() {
		if !sub.IsAvailableCommand() || sub.IsAdditionalHelpTopicCommand() {
			continue
		}
		if n := len(commandLabel(sub)); n > maxWidth {
			maxWidth = n
		}
	}
	return maxWidth
}

func rpad(s string, padding int) string {
	return fmt.Sprintf("%-*s", padding, s)
}

func trimTrailingWhitespaces(s string) string {
	return strings.TrimRight(s, " \t\r\n")
}

type helpStyler struct {
	enabled bool
}

func newHelpStyler(cmd *cobra.Command, w io.Writer) helpStyler {
	return helpStyler{enabled: helpColorEnabled(cmd, w)}
}

func (s helpStyler) section(label string) string {
	return s.render(styleSection, strings.ToUpper(label))
}

func (s helpStyler) commandName(label string) string {
	return s.render(styleCommandName, label)
}

// formatExamples renders the Example block with a 2-space indent. Comment
// lines (`# ...`) get comment styling; command lines get a `$` prompt
// prefix. Backslash line-continuations are honoured: a line ending in `\`
// marks the next non-blank line as a continuation, which is aligned under
// the command instead of receiving a fresh prompt.
func (s helpStyler) formatExamples(raw string) string {
	if raw == "" {
		return ""
	}
	lines := strings.Split(strings.TrimRight(raw, "\n"), "\n")
	out := make([]string, 0, len(lines))
	inContinuation := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			inContinuation = false
			out = append(out, "")
		case strings.HasPrefix(trimmed, "#"):
			inContinuation = false
			out = append(out, "  "+s.render(styleExampleComment, trimmed))
		default:
			if inContinuation {
				out = append(out, "    "+trimmed)
			} else {
				body := strings.TrimSpace(strings.TrimPrefix(trimmed, "$"))
				out = append(out, "  "+s.render(stylePrompt, "$")+" "+body)
			}
			inContinuation = strings.HasSuffix(trimmed, `\`)
		}
	}
	return strings.Join(out, "\n")
}

func (s helpStyler) render(style lipgloss.Style, value string) string {
	if !s.enabled {
		return value
	}
	return style.Render(value)
}

var (
	styleSection        = lipgloss.NewStyle().Bold(true)
	styleCommandName    = lipgloss.NewStyle()
	styleExampleComment = lipgloss.NewStyle()
	stylePrompt         = lipgloss.NewStyle()
)

func helpColorEnabled(cmd *cobra.Command, w io.Writer) bool {
	mode := "auto"
	if flag := cmd.Flag("color"); flag != nil {
		mode = flag.Value.String()
	}
	switch mode {
	case "never":
		return false
	case "always":
		return true
	}
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd())
}

func groupedCommands(cmd *cobra.Command) []commandGroupView {
	out := make([]commandGroupView, 0, len(cmd.Groups()))
	for _, g := range cmd.Groups() {
		view := commandGroupView{Title: g.Title}
		for _, sub := range cmd.Commands() {
			if !sub.IsAvailableCommand() || sub.IsAdditionalHelpTopicCommand() {
				continue
			}
			if sub.GroupID == g.ID {
				view.Commands = append(view.Commands, sub)
			}
		}
		if len(view.Commands) > 0 {
			out = append(out, view)
		}
	}
	return out
}

func ungroupedCommands(cmd *cobra.Command) []*cobra.Command {
	var out []*cobra.Command
	for _, sub := range cmd.Commands() {
		if !sub.IsAvailableCommand() || sub.IsAdditionalHelpTopicCommand() {
			continue
		}
		if sub.GroupID == "" {
			out = append(out, sub)
		}
	}
	return out
}
