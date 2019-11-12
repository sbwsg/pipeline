package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

// Validate that Artifact types implement Defaultable
var _ apis.Defaultable = (*ArtifactType)(nil)
var _ apis.Defaultable = (*ArtifactInstance)(nil)

// Validate that Artifact types implement Validatable
var _ apis.Validatable = (*ArtifactType)(nil)
var _ apis.Validatable = (*ArtifactInstance)(nil)

// ArtifactSpecMode is a string type indicating the "read-write" mode of
// an artifact type implementation.
type ArtifactSpecMode string

const (
	ArtifactROMode     ArtifactSpecMode = "ro"
	ArtifactRWMode     ArtifactSpecMode = "rw"
	ArtifactCreateMode ArtifactSpecMode = "create"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ArtifactType describe the steps, sidecars, parameters, workspaces, and resource results
// that will be added to a TaskRun using a resource of this type.
//
// +k8s:openapi-gen=true
type ArtifactType struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec ArtifactTypeSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ArtifactTypeList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArtifactType `json:"items"`
}

// ArtifactTypeSpec is the spec field for an ArtifactType
type ArtifactTypeSpec struct {
	// ReadOnlyMode declares an ArtifactImplementation that provides read-only
	// access to the contents of the artifact. Tasks using an artifact type in
	// this mode should not expect changes made to this artifact during the
	// task's execution to be reflected in the external system that this
	// artifact type works with.
	ReadOnlyMode *ArtifactImplementation `json:"readOnlyMode"`

	// ReadWriteMode is an ArtifactImplementation that provides read-write
	// access to the contents of the artifact. Any artifact type implementing
	// this mode is expected to sync changes made to this artifact during a
	// Task with the external system that this artifact type works with.
	ReadWriteMode *ArtifactImplementation `json:"readWriteMode"`

	// CreateMode is a declaration of a contract that a Task must fulfill
	// in order to correctly generate an artifact of this type.
	CreateMode *ArtifactContract `json:"createMode"`
}

// ArtifactContract describes the parameters and return values of a given artifact type
// in a given mode.
type ArtifactContract struct {
	// The paramaters that this artifact type takes.
	// +optional
	Params []ArtifactTypeParam `json:"params"`

	// The set of Resource Results that will be written out by an
	// artifact of this type and mode.
	// +optional
	Results []ArtifactTypeResult `json:"results"`
}

// TODO(sbws): description.
type ArtifactImplementation struct {
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

type ArtifactTypeResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

//
type ArtifactTypeParam struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Default     string `json:"default"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TODO(sbws): ArtifactInstance ...
type ArtifactInstance struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Spec ArtifactInstanceSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TODO(sbws): ArtifactInstanceList ...
type ArtifactInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArtifactInstance `json:"items"`
}

//
type ArtifactInstanceParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

//
type ArtifactInstanceSpec struct {
	Type   string                  `json:"type"`
	Params []ArtifactInstanceParam `json:"params"`
}

type ArtifactInstanceEmbedding struct {
	Name   string                  `json:"name"`
	Params []ArtifactInstanceParam `json:"params,omitempty"`
}

// ArtifactRequest is an entry in a Task definition requesting a pipeline resource
// of a specific type in a specific mode.
type ArtifactRequest struct {
	Name string           `json:"name"`
	Type string           `json:"type"`
	Mode ArtifactSpecMode `json:"mode"`
}
