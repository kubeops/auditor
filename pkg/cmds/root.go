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
	"os"

	"kmodules.xyz/auditor/apis/auditor/install"
	"kmodules.xyz/client-go/logs"
	"kmodules.xyz/client-go/tools/cli"

	"github.com/spf13/cobra"
	"gomodules.xyz/x/flags"
	v "gomodules.xyz/x/version"
	genericapiserver "k8s.io/apiserver/pkg/server"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
)

func NewRootCmd() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:               "auditor [command]",
		Short:             `Kubernetes Auditor by AppsCode`,
		DisableAutoGenTag: true,
		PersistentPreRun: func(c *cobra.Command, args []string) {
			flags.DumpAll(c.Flags())
			cli.SendAnalytics(c, v.Version.Version)

			install.Install(clientsetscheme.Scheme)
		},
	}
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	logs.ParseFlags()
	rootCmd.PersistentFlags().BoolVar(&cli.EnableAnalytics, "enable-analytics", cli.EnableAnalytics, "Send analytical events to Google Analytics")

	rootCmd.AddCommand(v.NewCmdVersion())
	stopCh := genericapiserver.SetupSignalHandler()
	rootCmd.AddCommand(NewCmdRun(os.Stdout, os.Stderr, stopCh))

	return rootCmd
}
