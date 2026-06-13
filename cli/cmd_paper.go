package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/openreview-cli/openreview"
)

func (a *App) paperCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "paper <id>",
		Short: "Show a single paper by OpenReview note ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.progressf("fetching paper %s...", args[0])
			p, err := a.client.Note(cmd.Context(), args[0])
			if err != nil {
				return mapFetchErr(err)
			}
			return a.render([]openreview.Paper{p})
		},
	}
}
