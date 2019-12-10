package v1alpha1

import "context"

func (p *Plugin) SetDefaults(ctx context.Context) {
	p.Spec.SetDefaults(ctx)
}

func (spec *PluginSpec) SetDefaults(ctx context.Context) {
}

func (a *Artifact) SetDefaults(ctx context.Context) {
	a.Spec.SetDefaults(ctx)
}

func (spec *ArtifactSpec) SetDefaults(ctx context.Context) {
}
