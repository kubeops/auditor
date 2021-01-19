package cmds

import (
	"flag"

	"kubeshield.dev/auditor/pkg/cmds/usage"

	"github.com/spf13/cobra"
	"gomodules.xyz/x/log"
	"kmodules.xyz/client-go/logs"
)

func NewCmdUsage() *cobra.Command {
	opts := usage.NewNatsOptions()
	var usageCmd = &cobra.Command{
		Use:               "usage",
		Short:             `Generate usage for billing and monitoring`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Infof("Generating usage for billing and monitoring ...")
			opts.StartNatsSubscription()

			return nil
		},
	}
	usageCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	logs.ParseFlags()

	opts.AddFlags(usageCmd.Flags())

	return usageCmd
}
