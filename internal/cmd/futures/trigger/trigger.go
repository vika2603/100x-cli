// Package trigger wires the `100x futures trigger` cobra verbs.
//
// `place` is a standalone trigger (StopOrderType=0); `attach order` and
// `attach position` use the leg-preserve helpers in shared/legs.go to honour
// the gateway's "send both SL and TP together" requirement without
// clobbering the leg the caller did not specify.
package trigger

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/internal/cmd/factory"
)

// NewCmdTrigger returns the `trigger` group.
func NewCmdTrigger(f *factory.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "trigger",
		Short: "Condition-order (SL / TP) operations",
	}
	c.AddCommand(
		NewCmdPlace(f),
		NewCmdAttach(f),
		NewCmdList(f),
		NewCmdEdit(f),
		NewCmdCancel(f),
		NewCmdCancelAll(f),
	)
	return c
}
