// Package trigger wires the `100x futures trigger` cobra verbs.
//
// `place` is a standalone trigger (StopOrderType=0); `attach order` and
// `attach position` use the side-preserve helpers in protection/ to honour
// the gateway's "send both SL and TP together" requirement without
// clobbering the side the caller did not specify.
package trigger

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/internal/cmd/factory"
)

// NewCmdTrigger returns the `trigger` group.
func NewCmdTrigger(f *factory.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:     "trigger",
		Aliases: []string{"t"},
		Short:   "Condition-order (SL / TP) operations",
		Long: "Manage standalone trigger orders and attached SL/TP.\n\n" +
			"`trigger place` creates a standalone conditional order. `trigger attach` manages\n" +
			"stop-loss and take-profit on existing orders or positions. `trigger list`, `edit`,\n" +
			"and `cancel` inspect or mutate pending trigger state.\n\n" +
			"Attached order-level SL/TP has backend constraints because the gateway applies\n" +
			"them at position scope. The command-specific help explains what the CLI preserves\n" +
			"automatically and when you must intervene manually.",
		Example: "# List active triggers for BTCUSDT\n" +
			"  100x futures trigger list BTCUSDT\n\n" +
			"# Place a standalone BUY trigger on BTCUSDT at trigger price 65000\n" +
			"  100x futures trigger place BTCUSDT --side buy --trigger-price 65000 --size 0.001\n\n" +
			"# Attach SL and TP together to an existing BTCUSDT order\n" +
			"  100x futures trigger attach order BTCUSDT <order-id> --sl-price 68000 --tp-price 82000",
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
