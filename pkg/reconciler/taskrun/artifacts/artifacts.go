package artifacts

import (
	"fmt"

	"golang.org/x/xerrors"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/names"
)

func ResolveArtifacts(tr *v1alpha1.TaskRun, ts *v1alpha1.TaskSpec, lister listers.PluginLister) error {
	if ts.Artifacts != nil {
		for _, a := range ts.Artifacts {
			name := a.Name
			typ := a.Type
			mode := a.Mode

			found := false

			if mode == "create" {
				return nil
			}

			for _, b := range tr.Spec.Artifacts {
				if b.Name != name {
					continue
				}
				artifactType, err := lister.Plugins(tr.Namespace).Get(typ)
				if err != nil {
					return xerrors.Errorf("error fetching artifact type %q: %w", typ, err)
				}
				if artifactType == nil {
					return fmt.Errorf("no artifact type found with name %q", typ)
				}
				impl := getImplementationSupportingMode(artifactType.Spec, mode)
				if impl != nil {
					inject(tr, ts, impl, &b)
					found = true
				}
			}

			if found == false {
				return fmt.Errorf("artifact missing: %q not provided by taskrun %q", name, tr.Name)
			}
		}
	}
	return nil
}

func inject(tr *v1alpha1.TaskRun, ts *v1alpha1.TaskSpec, impl *v1alpha1.PluginImplementation, artifactInstance *v1alpha1.ArtifactEmbedding) {
	if len(impl.Sidecars) > 0 {
		ts.Sidecars = append(ts.Sidecars, impl.Sidecars...)
	}
	if len(impl.PreRunSteps) > 0 {
		steps := append([]v1alpha1.Step{}, impl.PreRunSteps...)
		// Multiple resources of the same type will have step names that are the same so randomize their naming a bit
		for i := range steps {
			steps[i].Name = names.SimpleNameGenerator.RestrictLengthWithRandomSuffix(steps[i].Name)
		}
		rewriteParams(steps, impl, artifactInstance)
		steps = append(steps, ts.Steps...)
		ts.Steps = steps
	}
	if len(impl.PostRunSteps) > 0 {
		steps := append([]v1alpha1.Step{}, impl.PostRunSteps...)
		// Multiple resources of the same type will have step names that are the same so randomize their naming a bit
		for i := range steps {
			steps[i].Name = names.SimpleNameGenerator.RestrictLengthWithRandomSuffix(steps[i].Name)
		}
		rewriteParams(steps, impl, artifactInstance)
		ts.Steps = append(ts.Steps, steps...)
	}
}

// Replace $(params.foo) with $(artifacts.artifact-name.params.foo) and also $(name) with $(artifacts.artifact-name.name).
// Then some later function can do the actual work of replacing the variable with the value.
func rewriteParams(steps []v1alpha1.Step, impl *v1alpha1.PluginImplementation, inst *v1alpha1.ArtifactEmbedding) {
	replacements := make(map[string]string)
	for _, p := range impl.Params {
		key := "params." + p.Name
		value := fmt.Sprintf("$(artifacts.%s.params.%s)", inst.Name, p.Name)
		replacements[key] = value
	}
	replacements["name"] = fmt.Sprintf("$(artifacts.%s.name)", inst.Name)
	for i := range steps {
		v1alpha1.ApplyStepReplacements(&steps[i], replacements, nil)
	}
}

func getImplementationSupportingMode(spec v1alpha1.PluginSpec, desiredMode v1alpha1.PluginSpecMode) *v1alpha1.PluginImplementation {
	switch {
	case desiredMode == v1alpha1.PluginROMode && spec.ReadOnlyMode != nil:
		return spec.ReadOnlyMode
	case desiredMode == v1alpha1.PluginRWMode && spec.ReadWriteMode != nil:
		return spec.ReadWriteMode
	default:
	}
	return nil
}
