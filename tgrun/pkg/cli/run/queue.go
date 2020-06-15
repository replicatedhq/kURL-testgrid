package run

import (
	"github.com/replicatedhq/kurl/testgrid/tgrun/pkg/scheduler"
	schedulertypes "github.com/replicatedhq/kurl/testgrid/tgrun/pkg/scheduler/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func QueueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "queue",
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			schedulerOptions := schedulertypes.SchedulerOptions{
				APIEndpoint:  v.GetString("testgrid-api"),
				OverwriteRef: v.GetBool("overwrite-ref"),
				Ref:          v.GetString("ref"),
			}

			if err := scheduler.Run(schedulerOptions); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().String("ref", "", "ref to report to testgrid")
	cmd.Flags().String("testgrid-api", "https://api.testgrid.kurl.sh", "set to change the location of the testgrid api")
	cmd.Flags().Bool("overwrite-ref", false, "when set, overwrite the ref on the testgrid")

	cmd.MarkFlagRequired("ref")

	return cmd
}
