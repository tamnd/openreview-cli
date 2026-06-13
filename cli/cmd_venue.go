package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) venueCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "venue <venue-id>",
		Short: "List papers at an OpenReview venue",
		Long: `List papers submitted to a venue. Uses the /-/Blind_Submission invitation
and falls back to /-/Submission if no results are found.

Examples:
  orv venue ICLR.cc/2024/Conference
  orv venue NeurIPS.cc/2024/Conference -n 50`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			venueID := args[0]
			n := a.effectiveLimit(20)
			a.progressf("fetching papers for %s...", venueID)

			inv := venueID + "/-/Blind_Submission"
			papers, count, err := a.client.Notes(cmd.Context(), inv, n, 0)
			if err != nil {
				return mapFetchErr(err)
			}
			if count == 0 {
				inv = venueID + "/-/Submission"
				papers, _, err = a.client.Notes(cmd.Context(), inv, n, 0)
				if err != nil {
					return mapFetchErr(err)
				}
			}
			return a.renderOrEmpty(papers, len(papers))
		},
	}
}
