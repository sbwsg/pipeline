/*
Copyright 2019 The Tekton Authors

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

package config

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/ghodss/yaml"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	corev1 "k8s.io/api/core/v1"
)

const (
	DefaultTimeoutMinutes          = 60
	NoTimeoutDuration              = 0 * time.Minute
	defaultTimeoutMinutesKey       = "default-timeout-minutes"
	defaultServiceAccountKey       = "default-service-account"
	defaultManagedByLabelValueKey  = "default-managed-by-label-value"
	DefaultManagedByLabelValue     = "tekton-pipelines"
	defaultPodTemplateKey          = "default-pod-template"
	defaultTaskRunWorkspaceBinding = "default-task-run-workspace-binding"
)

// Defaults holds the default configurations
// +k8s:deepcopy-gen=true
type Defaults struct {
	DefaultTimeoutMinutes          int
	DefaultServiceAccount          string
	DefaultManagedByLabelValue     string
	DefaultPodTemplate             *pod.Template
	DefaultTaskRunWorkspaceBinding *WorkspaceBinding
}

type WorkspaceBinding struct {
	// SubPath is optionally a directory on the volume which should be used
	// for this binding (i.e. the volume will be mounted at this sub directory).
	// +optional
	SubPath string `json:"subPath,omitempty"`
	// VolumeClaimTemplate is a template for a claim that will be created in the same namespace.
	// The PipelineRun controller is responsible for creating a unique claim for each instance of PipelineRun.
	// +optional
	VolumeClaimTemplate *corev1.PersistentVolumeClaim `json:"volumeClaimTemplate,omitempty"`
	// PersistentVolumeClaimVolumeSource represents a reference to a
	// PersistentVolumeClaim in the same namespace. Either this OR EmptyDir can be used.
	// +optional
	PersistentVolumeClaim *corev1.PersistentVolumeClaimVolumeSource `json:"persistentVolumeClaim,omitempty"`
	// EmptyDir represents a temporary directory that shares a Task's lifetime.
	// More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir
	// Either this OR PersistentVolumeClaim can be used.
	// +optional
	EmptyDir *corev1.EmptyDirVolumeSource `json:"emptyDir,omitempty"`
	// ConfigMap represents a configMap that should populate this workspace.
	// +optional
	ConfigMap *corev1.ConfigMapVolumeSource `json:"configMap,omitempty"`
	// Secret represents a secret that should populate this workspace.
	// +optional
	Secret *corev1.SecretVolumeSource `json:"secret,omitempty"`
}

func (b *WorkspaceBinding) Equals(other *WorkspaceBinding) bool {
	if b == nil && other == nil {
		return true
	}
	if b == nil || other == nil {
		return false
	}
	return reflect.DeepEqual(b, other)
}

// GetDefaultsConfigName returns the name of the configmap containing all
// defined defaults.
func GetDefaultsConfigName() string {
	if e := os.Getenv("CONFIG_DEFAULTS_NAME"); e != "" {
		return e
	}
	return "config-defaults"
}

// Equals returns true if two Configs are identical
func (cfg *Defaults) Equals(other *Defaults) bool {
	if cfg == nil && other == nil {
		return true
	}

	if cfg == nil || other == nil {
		return false
	}

	return other.DefaultTimeoutMinutes == cfg.DefaultTimeoutMinutes &&
		other.DefaultServiceAccount == cfg.DefaultServiceAccount &&
		other.DefaultManagedByLabelValue == cfg.DefaultManagedByLabelValue &&
		other.DefaultPodTemplate.Equals(cfg.DefaultPodTemplate) &&
		other.DefaultTaskRunWorkspaceBinding.Equals(cfg.DefaultTaskRunWorkspaceBinding)
}

// NewDefaultsFromMap returns a Config given a map corresponding to a ConfigMap
func NewDefaultsFromMap(cfgMap map[string]string) (*Defaults, error) {
	tc := Defaults{
		DefaultTimeoutMinutes:      DefaultTimeoutMinutes,
		DefaultManagedByLabelValue: DefaultManagedByLabelValue,
	}

	if defaultTimeoutMin, ok := cfgMap[defaultTimeoutMinutesKey]; ok {
		timeout, err := strconv.ParseInt(defaultTimeoutMin, 10, 0)
		if err != nil {
			return nil, fmt.Errorf("failed parsing tracing config %q", defaultTimeoutMinutesKey)
		}
		tc.DefaultTimeoutMinutes = int(timeout)
	}

	if defaultServiceAccount, ok := cfgMap[defaultServiceAccountKey]; ok {
		tc.DefaultServiceAccount = defaultServiceAccount
	}

	if defaultManagedByLabelValue, ok := cfgMap[defaultManagedByLabelValueKey]; ok {
		tc.DefaultManagedByLabelValue = defaultManagedByLabelValue
	}

	if defaultPodTemplate, ok := cfgMap[defaultPodTemplateKey]; ok {
		var podTemplate pod.Template
		if err := yaml.Unmarshal([]byte(defaultPodTemplate), &podTemplate); err != nil {
			return nil, fmt.Errorf("failed to unmarshal %v", defaultPodTemplate)
		}
		tc.DefaultPodTemplate = &podTemplate
	}

	if bindingYAML, ok := cfgMap[defaultTaskRunWorkspaceBinding]; ok {
		var wb WorkspaceBinding
		if err := yaml.Unmarshal([]byte(bindingYAML), &wb); err != nil {
			return nil, fmt.Errorf("failed to unmarshal %v", bindingYAML)
		}
		tc.DefaultTaskRunWorkspaceBinding = &wb
	}

	return &tc, nil
}

// NewDefaultsFromConfigMap returns a Config for the given configmap
func NewDefaultsFromConfigMap(config *corev1.ConfigMap) (*Defaults, error) {
	return NewDefaultsFromMap(config.Data)
}
