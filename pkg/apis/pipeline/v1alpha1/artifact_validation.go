package v1alpha1

import (
	"context"

	"knative.dev/pkg/apis"
)

func (a ArtifactType) Validate(ctx context.Context) *apis.FieldError {
	if err := validateObjectMetadata(a.GetObjectMeta()); err != nil {
		return err.ViaField("metadata")
	}
	return a.Spec.Validate(ctx).ViaField("Spec")
}

func (a ArtifactTypeSpec) Validate(ctx context.Context) *apis.FieldError {
	if a.ReadOnlyMode == nil && a.ReadWriteMode == nil && a.CreateMode == nil {
		return &apis.FieldError{
			Message: `expect at least 1 implementation to be specified in readOnlyMode, readWriteMode or createMode fields`,
			Paths:   []string{},
		}
	}
	if ferr := a.ReadOnlyMode.Validate(ctx); ferr != nil {
		return ferr
	}
	if ferr := a.ReadWriteMode.Validate(ctx); ferr != nil {
		return ferr
	}
	// if ferr := a.CreateMode.Validate(ctx); ferr != nil {
	// 	return ferr
	// }
	return nil
}

func (ai *ArtifactImplementation) Validate(ctx context.Context) *apis.FieldError {
	if ai == nil {
		return nil
	}
	return nil
}

var supportedModes []ArtifactSpecMode = []ArtifactSpecMode{ArtifactROMode, ArtifactRWMode, ArtifactCreateMode}

func isSupportedMode(m ArtifactSpecMode) bool {
	for _, sm := range supportedModes {
		if m == sm {
			return true
		}
	}
	return false
}

func (ai ArtifactInstance) Validate(ctx context.Context) *apis.FieldError {
	return nil
}
