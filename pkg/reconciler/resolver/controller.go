/*
Copyright 2021 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resolver

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	clustertaskinformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/clustertask"
	taskinformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/task"
	taskruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/taskrun"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

// NewController instantiates a new controller.Impl from knative.dev/pkg/controller
func NewController() func(context.Context, configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		logger := logging.FromContext(ctx)
		kubeclientset := kubeclient.Get(ctx)
		pipelineclientset := pipelineclient.Get(ctx)
		taskRunInformer := taskruninformer.Get(ctx)
		taskInformer := taskinformer.Get(ctx)
		clusterTaskInformer := clustertaskinformer.Get(ctx)

		lister := taskRunInformer.Lister()

		r := &Reconciler{
			KubeClientSet:     kubeclientset,
			PipelineClientSet: pipelineclientset,
			taskRunLister:     taskRunInformer.Lister(),
			taskLister:        taskInformer.Lister(),
			clusterTaskLister: clusterTaskInformer.Lister(),

			LeaderAwareFuncs: reconciler.LeaderAwareFuncs{
				// TODO: not sure what purpose this serves yet but get error
				// from knative on controller startup without it.
				PromoteFunc: func(bkt reconciler.Bucket, enq func(reconciler.Bucket, types.NamespacedName)) error {
					all, err := lister.List(labels.Everything())
					if err != nil {
						return err
					}
					for _, elt := range all {
						// TODO: Consider letting users specify a filter in options.
						enq(bkt, types.NamespacedName{
							Namespace: elt.GetNamespace(),
							Name:      elt.GetName(),
						})
					}
					return nil
				},
			},
		}
		configStore := config.NewStore(logger.Named("config-store"))
		configStore.WatchConfigs(cmw)
		r.configStore = configStore

		ctrType := reflect.TypeOf(r).Elem()
		ctrTypeName := fmt.Sprintf("%s.%s", ctrType.PkgPath(), ctrType.Name())
		ctrTypeName = strings.ReplaceAll(ctrTypeName, "/", ".")
		impl := controller.NewImpl(r, logger, ctrTypeName)

		logger.Info("Setting up resolver controller event handlers")
		taskRunInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: acceptResourcesWithUnpopulatedStatusSpec,
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc: impl.Enqueue,
				// UpdateFunc: controller.PassNew(impl.Enqueue),
			},
		})

		return impl
	}
}

func acceptResourcesWithUnpopulatedStatusSpec(obj interface{}) bool {
	tr, ok := obj.(*v1beta1.TaskRun)
	if !ok {
		return false
	}
	return tr.Status.TaskSpec == nil
}
