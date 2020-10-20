/*
Copyright 2020 The Tekton Authors

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

// nolint: golint
package v1alpha1

import (
	"context"
	"fmt"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"knative.dev/pkg/apis"
)

var _ apis.Convertible = (*PipelineRun)(nil)

// ConvertTo implements api.Convertible
func (source *PipelineRun) ConvertTo(ctx context.Context, obj apis.Convertible) error {
	switch sink := obj.(type) {
	case *v1beta1.PipelineRun:
		sink.ObjectMeta = source.ObjectMeta
		if err := source.Spec.ConvertTo(ctx, &sink.Spec); err != nil {
			return err
		}
		sink.Status = source.Status
		return nil
	default:
		return fmt.Errorf("unknown version, got: %T", sink)
	}
}

func (source *PipelineRunSpec) ConvertTo(ctx context.Context, sink *v1beta1.PipelineRunSpec) error {
	sink.PipelineRef = source.PipelineRef
	if source.PipelineSpec != nil {
		sink.PipelineSpec = &v1beta1.PipelineSpec{}
		if err := source.PipelineSpec.ConvertTo(ctx, sink.PipelineSpec); err != nil {
			return err
		}
	}
	sink.Resources = source.Resources
	sink.Params = source.Params
	sink.ServiceAccountName = source.ServiceAccountName
	sink.ServiceAccountNames = source.ServiceAccountNames
	sink.Status = source.Status
	sink.Timeout = source.Timeout
	sink.PodTemplate = source.PodTemplate
	if source.Workspaces != nil {
		sink.Workspaces = make([]v1beta1.PipelineRunWorkspaceBinding, len(source.Workspaces))
		for i := range sink.Workspaces {
			sink.Workspaces[i].WorkspaceBinding = source.Workspaces[i]
		}
	}
	return nil
}

// ConvertFrom implements api.Convertible
func (sink *PipelineRun) ConvertFrom(ctx context.Context, obj apis.Convertible) error {
	switch source := obj.(type) {
	case *v1beta1.PipelineRun:
		sink.ObjectMeta = source.ObjectMeta
		if err := sink.Spec.ConvertFrom(ctx, &source.Spec); err != nil {
			return err
		}
		sink.Status = source.Status
		return nil
	default:
		return fmt.Errorf("unknown version, got: %T", sink)
	}
}

func (sink *PipelineRunSpec) ConvertFrom(ctx context.Context, source *v1beta1.PipelineRunSpec) error {
	sink.PipelineRef = source.PipelineRef
	if source.PipelineSpec != nil {
		sink.PipelineSpec = &PipelineSpec{}
		if err := sink.PipelineSpec.ConvertFrom(ctx, *source.PipelineSpec); err != nil {
			return err
		}
	}
	sink.Resources = source.Resources
	sink.Params = source.Params
	sink.ServiceAccountName = source.ServiceAccountName
	sink.ServiceAccountNames = source.ServiceAccountNames
	sink.Status = source.Status
	sink.Timeout = source.Timeout
	sink.PodTemplate = source.PodTemplate
	if source.Workspaces != nil {
		sink.Workspaces = make([]v1beta1.WorkspaceBinding, len(source.Workspaces))
		for i := range sink.Workspaces {
			sink.Workspaces[i] = source.Workspaces[i].WorkspaceBinding
		}
	}
	return nil
}
