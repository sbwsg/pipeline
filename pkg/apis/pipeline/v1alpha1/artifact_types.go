package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

// Validate that Artifact types implement Defaultable
var _ apis.Defaultable = (*Plugin)(nil)
var _ apis.Defaultable = (*Artifact)(nil)

// Validate that Artifact types implement Validatable
var _ apis.Validatable = (*Plugin)(nil)
var _ apis.Validatable = (*Artifact)(nil)

// PluginSpecMode is a string type indicating the "read-write" mode of
// an artifact type implementation.
type PluginSpecMode string

const (
	PluginROMode     PluginSpecMode = "ro"
	PluginRWMode     PluginSpecMode = "rw"
	PluginCreateMode PluginSpecMode = "create"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Plugin describe the steps, sidecars, parameters, workspaces, and resource results
// that will be added to a TaskRun consuming or producing an artifact of this type.
//
// +k8s:openapi-gen=true
type Plugin struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec PluginSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PluginList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Plugin `json:"items"`
}

// PluginSpec is the spec field for a Plugin
type PluginSpec struct {
	// ReadOnlyMode declares a PluginImplementation that provides read-only
	// access to... Tasks using plugins in
	// this mode should not expect changes made to artifacts during the
	// task's execution to be reflected in the external system that this
	// plugin works with.
	ReadOnlyMode *PluginImplementation `json:"readOnlyMode"`

	// ReadWriteMode is an PluginImplementation that provides read-write
	// access to... Any plugin implementing
	// this mode is expected to sync changes made to artifacts during a
	// Task with the external system that this plugin works with.
	ReadWriteMode *PluginImplementation `json:"readWriteMode"`

	// CreateMode is a declaration of a contract that a Task must fulfill
	// in order to correctly generate an artifact of this type.
	CreateMode *ArtifactContract `json:"createMode"`
}

// ArtifactContract describes the parameters and return values of a given artifact type
// in a given mode.
type ArtifactContract struct {
	// The paramaters that this artifact type takes.
	// +optional
	Params []PluginParam `json:"params"`

	// The set of Resource Results that will be written out by an
	// artifact of this type and mode.
	// +optional
	Results []PluginResult `json:"results"`
}

// TODO(sbws): description.
type PluginImplementation struct {
	ArtifactContract

	// A human-readable description of what this implementation does.
	Description string `json:"description"`

	// Indicates whether this implementation
	// requires a storage mechanism like the existing Artifact
	// PVC / GCS implementation. The current use-case
	// for this is the FileSet artifact type.
	//
	// +optional
	Storage bool `json:"storage"`

	// The slice of steps that will be prepended to a TaskRun's steps.
	PreRunSteps []Step `json:"preRunSteps"`

	// The slice of steps that will be appended to a TaskRun's steps.
	PostRunSteps []Step `json:"postRunSteps"`

	// The slide of sidecar containers that will be added to a TaskRun's sidecars list.
	Sidecars []corev1.Container `json:"sidecars"`

	// TODO(sbws): do we need to introduce the "workspaces" concept?
}

type PluginResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

//
type PluginParam struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Default     string `json:"default"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TODO(sbws): Artifact ...
type Artifact struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Spec ArtifactSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TODO(sbws): ArtifactList ...
type ArtifactList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Artifact `json:"items"`
}

//
type ArtifactParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

//
type ArtifactSpec struct {
	Type   string          `json:"type"`
	Params []ArtifactParam `json:"params"`
}

type ArtifactEmbedding struct {
	Name   string          `json:"name"`
	Params []ArtifactParam `json:"params,omitempty"`
}

// ArtifactRequest is an entry in a Task definition requesting a pipeline resource
// of a specific type in a specific mode.
type ArtifactRequest struct {
	Name string         `json:"name"`
	Type string         `json:"type"`
	Mode PluginSpecMode `json:"mode"`
}
