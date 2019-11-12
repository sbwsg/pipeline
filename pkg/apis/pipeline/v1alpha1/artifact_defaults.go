package v1alpha1

import "context"

func (typ *ArtifactType) SetDefaults(ctx context.Context) {
	typ.Spec.SetDefaults(ctx)
}

func (spec *ArtifactTypeSpec) SetDefaults(ctx context.Context) {
}

func (inst *ArtifactInstance) SetDefaults(ctx context.Context) {
	inst.Spec.SetDefaults(ctx)
}

func (spec *ArtifactInstanceSpec) SetDefaults(ctx context.Context) {
}
