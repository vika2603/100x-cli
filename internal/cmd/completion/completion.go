// Package completion wires the cobra-provided shell-completion subcommand.
package completion

import (
	"os"

	"github.com/spf13/cobra"
)

// NewCmdCompletion returns `<root> completion <shell>`.
func NewCmdCompletion() *cobra.Command {
	return &cobra.Command{
		Use:                   "completion <bash|zsh|fish|powershell>",
		Short:                 "Generate a shell completion script",
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(c *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return c.Root().GenBashCompletionV2(os.Stdout, true)
			case "zsh":
				return c.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return c.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return c.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}
}
