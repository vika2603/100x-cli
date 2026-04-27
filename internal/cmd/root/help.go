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

const helpTemplate = `{{with or .Long .Short}}{{. | trimTrailingWhitespaces}}{{end}}

{{section "Usage"}}:
  {{.UseLine}}{{commandSuffix .}}
{{if .Aliases}}
{{section "Aliases"}}:
  {{join .Aliases ", "}}
{{end}}
{{if .Example}}
{{section "Examples"}}:
{{formatExamples .Example}}
{{end}}
{{if .HasAvailableSubCommands}}
{{section "Commands"}}:
{{if groupedCommands .}}{{range groupedCommands .}}{{if .Commands}}
  {{groupTitle .Title}}
{{range .Commands}}    {{commandName (rpad (commandLabel .) (commandNamePadding $)) }} {{.Short}}
{{end}}{{end}}{{end}}{{if ungroupedCommands .}}
  {{groupTitle "Additional Commands"}}
{{range ungroupedCommands .}}    {{commandName (rpad (commandLabel .) (commandNamePadding $)) }} {{.Short}}
{{end}}{{end}}{{else}}{{range ungroupedCommands .}}  {{commandName (rpad (commandLabel .) (commandNamePadding $)) }} {{.Short}}
{{end}}{{end}}
{{end}}{{if .HasAvailableLocalFlags}}
{{section "Flags"}}:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}{{if .HasAvailableInheritedFlags}}
{{section "Global Flags"}}:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}
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
		"groupTitle":              styler.groupTitle,
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
	max := 0
	for _, sub := range cmd.Commands() {
		if !sub.IsAvailableCommand() || sub.IsAdditionalHelpTopicCommand() {
			continue
		}
		if n := len(commandLabel(sub)); n > max {
			max = n
		}
	}
	return max
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

func (s helpStyler) groupTitle(label string) string {
	return s.render(styleGroupTitle, label)
}

func (s helpStyler) commandName(label string) string {
	return s.render(styleCommandName, label)
}

func (s helpStyler) formatExamples(raw string) string {
	if raw == "" {
		return ""
	}
	lines := strings.Split(strings.TrimRight(raw, "\n"), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			continue
		case strings.HasPrefix(trimmed, "#"):
			out = append(out, "  "+s.render(styleExampleComment, trimmed))
		default:
			trimmed = strings.TrimPrefix(trimmed, "$")
			trimmed = strings.TrimSpace(trimmed)
			out = append(out, "  "+s.render(stylePrompt, "$")+" "+trimmed)
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
	styleSection        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	styleGroupTitle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("8"))
	styleCommandName    = lipgloss.NewStyle().Bold(true)
	styleExampleComment = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	stylePrompt         = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
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
