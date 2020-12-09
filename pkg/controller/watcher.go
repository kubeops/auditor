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
	"strings"

	stringz "gomodules.xyz/x/strings"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	"kmodules.xyz/client-go/tools/clusterid"
)

func (c *AuditorController) initWatchers() error {
	cid, err := clusterid.ClusterUID(c.kubeClient.CoreV1().Namespaces())
	if err != nil {
		return err
	}
	fmt.Println("cluster id:", cid)

	disco := memory.NewMemCacheClient(c.kubeClient.Discovery())
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(disco)

	handler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			u, ok := obj.(*unstructured.Unstructured)
			if !ok {
				return
			}
			fmt.Println(u.GetObjectKind().GroupVersionKind(), u.GetUID(), u.GetName(), u.GetGeneration())
			//if data, err := yaml.Marshal(u); err == nil {
			//	fmt.Println(string(data))
			//}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			uOld, ok := oldObj.(*unstructured.Unstructured)
			if !ok {
				return
			}
			uNew, ok := newObj.(*unstructured.Unstructured)
			if !ok {
				return
			}
			if uOld.GetUID() == uNew.GetUID() && uOld.GetGeneration() == uNew.GetGeneration() {
				return
			}
			fmt.Println(uNew.GetObjectKind().GroupVersionKind(), uNew.GetUID(), uNew.GetName(), uNew.GetGeneration())
			//if data, err := yaml.Marshal(u); err == nil {
			//	fmt.Println(string(data))
			//}
		},
		DeleteFunc: func(obj interface{}) {
			if d, ok := obj.(cache.DeletedFinalStateUnknown); ok {
				fmt.Println(d.Key)
				return
			}
			u, ok := obj.(*unstructured.Unstructured)
			if !ok {
				return
			}
			fmt.Println(u.GetObjectKind().GroupVersionKind(), u.GetUID(), u.GetName(), u.GetGeneration())
		},
	}

	if len(c.Policy.Resources) == 0 {
		// watch all
		rsLists, err := disco.ServerPreferredResources()
		if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
			return err
		}
		for _, rsList := range rsLists {
			for _, rs := range rsList.APIResources {
				// skip sub resource
				if strings.ContainsRune(rs.Name, '/') {
					continue
				}
				// if resource can't be listed or read (get) skip it
				if !stringz.Contains(rs.Verbs, "list") || !stringz.Contains(rs.Verbs, "get") || !stringz.Contains(rs.Verbs, "watch") {
					continue
				}
				gv, err := schema.ParseGroupVersion(rsList.GroupVersion)
				if err != nil {
					return err
				}
				gvr := gv.WithResource(rs.Name)
				klog.Infoln("watching", gvr)
				c.dynamicInformerFactory.ForResource(gvr).Informer().AddEventHandler(handler)
			}
		}
	} else {
		for _, resource := range c.Policy.Resources {
			for _, name := range resource.Resources {
				if strings.ContainsRune(name, '/') {
					continue
				}
				gvr := schema.GroupVersionResource{
					Group: resource.Group,
					// Version:  "",
					Resource: name,
				}
				gvr, err := mapper.ResourceFor(gvr)
				if err != nil {
					return err
				}
				klog.Infoln("watching", gvr)
				c.dynamicInformerFactory.ForResource(gvr).Informer().AddEventHandler(handler)
			}
		}
	}
	return nil
}
