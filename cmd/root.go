package cmd

import (
	"github.com/spf13/cobra"
	"github.com/vitech-team/sdlcctl/cmd/promotion"
	"github.com/vitech-team/sdlcctl/cmd/topology"
	"github.com/vitech-team/sdlcctl/cmd/utils"
)

func Main() *cobra.Command {
	rootOpts := utils.Options{}
	cmd := &cobra.Command{
		Use:   "sdlc",
		Short: "SDLC utility commands",
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				panic(err.Error())
			}
		},
	}

	rootOpts.AddBaseFlags(cmd)

	topologyCmd, _ := topology.NewTopologyCmd(&rootOpts)
	cmd.AddCommand(topologyCmd)
	promotionCmd, _ := promotion.NewPromotionCmd(&rootOpts)
	cmd.AddCommand(promotionCmd)

	return cmd
}
