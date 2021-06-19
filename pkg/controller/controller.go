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

package controller

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
)

type AuditorController struct {
	config
	clientConfig *rest.Config

	kubeClient    kubernetes.Interface
	dynamicClient dynamic.Interface
	recorder      record.EventRecorder

	dynamicInformerFactory dynamicinformer.DynamicSharedInformerFactory
}

func (c *AuditorController) Run(stopCh <-chan struct{}) {
	if err := c.initWatchers(stopCh); err != nil {
		runtime.HandleError(err)
		return
	}
	go c.RunInformers(stopCh)

	<-stopCh
}

func (c *AuditorController) RunInformers(stopCh <-chan struct{}) {
	defer runtime.HandleCrash()

	klog.Info("Starting Auditor")

	c.dynamicInformerFactory.Start(stopCh)
	for _, v := range c.dynamicInformerFactory.WaitForCacheSync(stopCh) {
		if !v {
			runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
			return
		}
	}

	<-stopCh
	klog.Info("Stopping Auditor")
}
