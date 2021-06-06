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
	"strings"

	"go.bytebuilders.dev/audit/lib"
	stringz "gomodules.xyz/x/strings"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/klog/v2"
	disco_util "kmodules.xyz/client-go/discovery"
	dynamicfactory "kmodules.xyz/client-go/dynamic/factory"
	"kmodules.xyz/resource-metadata/pkg/graph"
)

func (c *AuditorController) initWatchers(stopCh <-chan struct{}) error {
	disco := c.kubeClient.Discovery()
	mapper := disco_util.NewResourceMapper(disco_util.NewRestMapper(disco))
	factory := dynamicfactory.NewSharedCached(c.dynamicInformerFactory, stopCh)
	//fn := lib.BillingEventCreator{
	//	Mapper: mapper,
	//	LicenseID: c.nats.LicenseID,
	//}
	g, err := graph.LoadGraphOfKnownResources()
	if err != nil {
		return err
	}
	fn2 := lib.AuditEventCreator{
		Graph: g,
		Finder: &graph.ObjectFinder{
			Factory: factory,
			Mapper:  mapper,
		},
		Factory:   factory,
		Mapper:    mapper,
		LicenseID: c.nats.LicenseID,
	}

	handler := lib.NewEventPublisher(
		c.nats,
		mapper,
		fn2.CreateEvent,
	)

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

				gvr, err := mapper.Preferred(gvr)
				if err != nil {
					klog.Errorln(err)
					continue
				}
				klog.Infoln("watching", gvr)
				c.dynamicInformerFactory.ForResource(gvr).Informer().AddEventHandler(handler)
			}
		}
	}
	return nil
}
