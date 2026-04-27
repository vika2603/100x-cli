package profile

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/internal/config"
)

// CompleteNames lists configured profile names for tab completion
// without making any network call.
func CompleteNames(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return completeNames(toComplete)
}

// CompleteNameFlag lists profile names for the global --profile flag.
func CompleteNameFlag(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return completeNames(toComplete)
}

func completeNames(toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		if toComplete != "" && !strings.HasPrefix(name, toComplete) {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out, cobra.ShellCompDirectiveNoFileComp
}
