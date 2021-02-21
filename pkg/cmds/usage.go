/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmds

import (
	"flag"

	"kmodules.xyz/auditor/pkg/cmds/usage"
	"kmodules.xyz/client-go/logs"

	"github.com/spf13/cobra"
	"gomodules.xyz/x/log"
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
