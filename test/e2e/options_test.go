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

package e2e_test

import (
	"os"
	"path/filepath"

	"kubeshield.dev/auditor/pkg/cmds/server"

	"k8s.io/client-go/util/homedir"
)

type E2EOptions struct {
	*server.ExtraOptions

	kubeContext string
	kubeConfig  string

	apiKey string
}

var (
	options = &E2EOptions{
		ExtraOptions: server.NewExtraOptions(),

		kubeConfig: func() string {
			kubecfg := os.Getenv("KUBECONFIG")
			if kubecfg != "" {
				return kubecfg
			}
			return filepath.Join(homedir.HomeDir(), ".kube", "config")
		}(),
		apiKey: "admin:admin",
	}
)
