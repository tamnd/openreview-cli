package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/openreview-cli/openreview"
)

func (a *App) topCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "top",
		Short: "List top ML conference venues on OpenReview",
		RunE: func(_ *cobra.Command, _ []string) error {
			venues := openreview.TopVenues
			return a.renderOrEmpty(venues, len(venues))
		},
	}
}
