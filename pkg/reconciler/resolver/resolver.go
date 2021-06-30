package resolver

import (
	"context"
	"fmt"
	"log"

	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	podconvert "github.com/tektoncd/pipeline/pkg/pod"
	"github.com/tektoncd/pipeline/pkg/reconciler/taskrun/resources"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

type Reconciler struct {
	// LeaderAwareFuncs is inlined to help us implement reconciler.LeaderAware
	reconciler.LeaderAwareFuncs

	KubeClientSet     kubernetes.Interface
	PipelineClientSet clientset.Interface
	taskRunLister     listers.TaskRunLister
	taskLister        listers.TaskLister
	clusterTaskLister listers.ClusterTaskLister
	configStore       reconciler.ConfigStore
}

var _ controller.Reconciler = &Reconciler{}

func (r *Reconciler) Reconcile(ctx context.Context, key string) error {
	if r.configStore != nil {
		ctx = r.configStore.ToContext(ctx)
	}

	logger := logging.FromContext(ctx)

	logger.Infof("HELLO. RECONCILER. Key: %v", key)

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return fmt.Errorf("invalid resource key %q: %v", key, err)
	}
	logger.Infof("NAMESPACE: %v, NAME: %v", namespace, name)

	tr, err := r.taskRunLister.TaskRuns(namespace).Get(name)
	if err != nil {
		err = fmt.Errorf("error getting taskrun for key %q: %w", key, err)
		tr.Status.MarkResourceFailed(podconvert.ReasonFailedResolution, err)
		logger.Error(err)
		return err
	}

	if tr.Status.TaskSpec != nil {
		return nil
	}

	getTaskFunc, err := resources.GetTaskFuncFromTaskRun(ctx, r.KubeClientSet, r.PipelineClientSet, tr)
	if err != nil {
		err = fmt.Errorf("error getting task func for taskrun %q: %w", key, err)
		tr.Status.MarkResourceFailed(podconvert.ReasonFailedResolution, err)
		logger.Error(err)
		return err
	}

	log.Println("\n\n\nGETTING TASK DATA!\n\n\n")

	taskMeta, taskSpec, err := resources.GetTaskData(ctx, tr, getTaskFunc)

	log.Println("\n\n\nGETTASKDATA RETURNED!\n\n\n")

	if err != nil {
		err = fmt.Errorf("error getting task func for taskrun %q: %w", key, err)
		tr.Status.MarkResourceFailed(podconvert.ReasonFailedResolution, err)
		logger.Error(err)
		return err
	} else if taskSpec == nil {
		err = fmt.Errorf("no task found for taskrun %q", key)
		tr.Status.MarkResourceFailed(podconvert.ReasonCouldntGetTask, err)
		logger.Error(err)
		return err
	}

	if taskSpec != nil {
		tr.Status.TaskSpec = taskSpec
	}

	if tr.ObjectMeta.Labels == nil && len(taskMeta.Labels) > 0 {
		tr.ObjectMeta.Labels = map[string]string{}
	}
	for key, value := range taskMeta.Labels {
		tr.ObjectMeta.Labels[key] = value
	}

	_, err = r.PipelineClientSet.TektonV1beta1().TaskRuns(tr.Namespace).UpdateStatus(ctx, tr, metav1.UpdateOptions{})
	if err != nil {
		err = fmt.Errorf("error updating taskrun %q with task spec: %w", key, err)
		tr.Status.MarkResourceFailed(podconvert.ReasonFailedResolution, err)
		logger.Error(err)
		return err
	}

	return nil
}
