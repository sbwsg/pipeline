package artifacts

import (
	"fmt"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
)

func ProcessPipeline(spec *v1alpha1.PipelineSpec, run *v1alpha1.PipelineRun) error {
	if spec == nil || run == nil {
		return fmt.Errorf("nil pipeline spec or run received")
	}

	if err := ensurePipelineRunProvidesArtifactsExpectedByPipelineSpec(spec, run); err != nil {
		return err
	}

	// Validate From clauses.
	// a) They must exist
	// b) They must refer to the pipeline spec's artifacts or
	// c) They must refer to a task's RW or Create artifacts
	//    - need tasks to be resolved completely for this to work.
	for _, pt := range spec.Tasks {
		for _, a := range pt.Artifacts {
			if strings.TrimSpace(a.From) == "" {
				return fmt.Errorf("task %q must indicate where artifact %q is from", fmt.Sprintf("%s/%s", run.Namespace, run.Name), a.Name)
			}

			parts := strings.Split(a.From, ".")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			topLevelSection := parts[0]
			if topLevelSection == "artifacts" {
				if err := ensurePipelineSpecProvidesArtifact(spec, parts[1]); err != nil {
					return fmt.Errorf("task %q artifact error: %w", pt.Name, err)
				}
			} else if topLevelSection == "tasks" {
				// TODO: ensure task exists that declares an RW or Create artifact
				// need the fully-hydrated task for this to work :/
			}
		}
	}
	return nil
}

// Validate that artifacts in spec's Artifacts section have matching artifacts
// in the pipeline run.
func ensurePipelineRunProvidesArtifactsExpectedByPipelineSpec(spec *v1alpha1.PipelineSpec, run *v1alpha1.PipelineRun) error {
	missingArtifactNames := []string{}
CHECK_FOR_MISSING_ARTIFACTS:
	for _, specArtifact := range spec.Artifacts {
		for _, runArtifact := range run.Spec.Artifacts {
			if specArtifact.Name == runArtifact.Name {
				continue CHECK_FOR_MISSING_ARTIFACTS
			}
		}
		missingArtifactNames = append(missingArtifactNames, specArtifact.Name)
	}

	if len(missingArtifactNames) > 0 {
		return fmt.Errorf("artifacts expected by pipeline spec are missing from pipeline run %s: %s", fmt.Sprintf("%s/%s", run.Namespace, run.Name), strings.Join(missingArtifactNames, ", "))
	}
	return nil
}

func ensurePipelineSpecProvidesArtifact(spec *v1alpha1.PipelineSpec, expectedArtifactName string) error {
	for _, specArtifact := range spec.Artifacts {
		if specArtifact.Name == expectedArtifactName {
			return nil
		}
	}
	return fmt.Errorf("pipeline does not provide artifact %q in top-level artifacts section", expectedArtifactName)
}
