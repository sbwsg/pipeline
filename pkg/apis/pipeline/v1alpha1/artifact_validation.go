package v1alpha1

import (
	"context"

	"knative.dev/pkg/apis"
)

func (p Plugin) Validate(ctx context.Context) *apis.FieldError {
	if err := validateObjectMetadata(p.GetObjectMeta()); err != nil {
		return err.ViaField("metadata")
	}
	return p.Spec.Validate(ctx).ViaField("Spec")
}

func (p PluginSpec) Validate(ctx context.Context) *apis.FieldError {
	if p.ReadOnlyMode == nil && p.ReadWriteMode == nil && p.CreateMode == nil {
		return &apis.FieldError{
			Message: `expect at least 1 implementation to be specified in readOnlyMode, readWriteMode or createMode fields`,
			Paths:   []string{},
		}
	}
	if ferr := p.ReadOnlyMode.Validate(ctx); ferr != nil {
		return ferr
	}
	if ferr := p.ReadWriteMode.Validate(ctx); ferr != nil {
		return ferr
	}
	// if ferr := a.CreateMode.Validate(ctx); ferr != nil {
	// 	return ferr
	// }
	return nil
}

func (p *PluginImplementation) Validate(ctx context.Context) *apis.FieldError {
	if p == nil {
		return nil
	}
	return nil
}

var supportedModes []PluginSpecMode = []PluginSpecMode{PluginROMode, PluginRWMode, PluginCreateMode}

func isSupportedMode(m PluginSpecMode) bool {
	for _, sm := range supportedModes {
		if m == sm {
			return true
		}
	}
	return false
}

func (a Artifact) Validate(ctx context.Context) *apis.FieldError {
	return nil
}
