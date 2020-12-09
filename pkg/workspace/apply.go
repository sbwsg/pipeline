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
package workspace

import (
	"fmt"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/names"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	volumeNameBase = "ws"
)

// nameVolumeMap is a map from a workspace's name to its Volume.
type nameVolumeMap map[string]corev1.Volume

// setVolumeSource assigns a volume to a workspace's name.
func (nvm nameVolumeMap) setVolumeSource(workspaceName string, volumeName string, source corev1.VolumeSource) {
	nvm[workspaceName] = corev1.Volume{
		Name:         volumeName,
		VolumeSource: source,
	}
}

// CreateVolumes will return a dictionary where the keys are the names of the workspaces bound in
// wb and the value is a newly-created Volume to use. If the same Volume is bound twice, the
// resulting volumes will both have the same name to prevent the same Volume from being attached
// to a pod twice. The names of the returned volumes will be a short random string starting "ws-".
func CreateVolumes(wb []v1beta1.WorkspaceBinding) map[string]corev1.Volume {
	pvcs := map[string]corev1.Volume{}
	v := make(nameVolumeMap)
	for _, w := range wb {
		name := names.SimpleNameGenerator.RestrictLengthWithRandomSuffix(volumeNameBase)
		switch {
		case w.PersistentVolumeClaim != nil:
			// If it's a PVC, we need to check if we've encountered it before so we avoid mounting it twice
			if vv, ok := pvcs[w.PersistentVolumeClaim.ClaimName]; ok {
				v[w.Name] = vv
			} else {
				pvc := *w.PersistentVolumeClaim
				v.setVolumeSource(w.Name, name, corev1.VolumeSource{PersistentVolumeClaim: &pvc})
				pvcs[pvc.ClaimName] = v[w.Name]
			}
		case w.EmptyDir != nil:
			ed := *w.EmptyDir
			v.setVolumeSource(w.Name, name, corev1.VolumeSource{EmptyDir: &ed})
		case w.ConfigMap != nil:
			cm := *w.ConfigMap
			v.setVolumeSource(w.Name, name, corev1.VolumeSource{ConfigMap: &cm})
		case w.Secret != nil:
			s := *w.Secret
			v.setVolumeSource(w.Name, name, corev1.VolumeSource{Secret: &s})
		}
	}
	return v
}

func getDeclaredWorkspace(name string, w []v1beta1.WorkspaceDeclaration) (*v1beta1.WorkspaceDeclaration, error) {
	for _, workspace := range w {
		if workspace.Name == name {
			return &workspace, nil
		}
	}
	// Trusting validation to ensure
	return nil, fmt.Errorf("even though validation should have caught it, bound workspace %s did not exist in declared workspaces", name)
}

// Apply updates the StepTemplate, Sidecars and Volumes declaration in ts so that workspaces
// specified through wb combined with the declared workspaces in ts will be available for
// all containers in the resulting pod.
func Apply(ts v1beta1.TaskSpec, wb []v1beta1.WorkspaceBinding, v map[string]corev1.Volume) (*v1beta1.TaskSpec, error) {
	// If there are no bound workspaces, we don't need to do anything
	if len(wb) == 0 {
		return &ts, nil
	}

	addedVolumes := sets.NewString()

	// Initialize StepTemplate if it hasn't been already
	if ts.StepTemplate == nil {
		ts.StepTemplate = &corev1.Container{}
	}

	shared := map[string]v1beta1.WorkspaceBinding{}
	for _, w := range wb {
		shared[w.Name] = w
	}

	exclusives := []string{}
	for i := range ts.Steps {
		step := &ts.Steps[i]
		for _, ws := range step.Workspaces {
			bind, ok := shared[ws.Name]
			if !ok {
				return nil, fmt.Errorf("No binding found for Workspace %q referenced by Step %d (%q)", ws.Name, i, step.Name)
			}
			exclusives = append(exclusives, ws.Name)
			decl, err := getDeclaredWorkspace(ws.Name, ts.Workspaces)
			if err != nil {
				return nil, err
			}
			vol := v[ws.Name]
			if !addedVolumes.Has(vol.Name) {
				ts.Volumes = append(ts.Volumes, vol)
				addedVolumes.Insert(vol.Name)
			}
			volumeMount := workspaceVolumeMount(vol, *decl, bind)
			step.VolumeMounts = append(step.VolumeMounts, volumeMount)
		}
	}

	for i := range ts.Sidecars {
		sidecar := &ts.Sidecars[i]
		for _, ws := range sidecar.Workspaces {
			bind, ok := shared[ws.Name]
			if !ok {
				return nil, fmt.Errorf("No binding found for Workspace %q referenced by Sidecar %d (%q)", ws.Name, i, sidecar.Name)
			}
			exclusives = append(exclusives, ws.Name)
			decl, err := getDeclaredWorkspace(ws.Name, ts.Workspaces)
			if err != nil {
				return nil, err
			}
			vol := v[ws.Name]
			if !addedVolumes.Has(vol.Name) {
				ts.Volumes = append(ts.Volumes, vol)
				addedVolumes.Insert(vol.Name)
			}
			volumeMount := workspaceVolumeMount(vol, *decl, bind)
			sidecar.VolumeMounts = append(sidecar.VolumeMounts, volumeMount)
		}
	}

	for _, exclusive := range exclusives {
		delete(shared, exclusive)
	}

	for bindName, bind := range shared {
		decl, err := getDeclaredWorkspace(bindName, ts.Workspaces)
		if err != nil {
			return nil, err
		}
		vol := v[bindName]
		if !addedVolumes.Has(vol.Name) {
			ts.Volumes = append(ts.Volumes, vol)
			addedVolumes.Insert(vol.Name)
		}
		volumeMount := workspaceVolumeMount(vol, *decl, bind)
		ts.StepTemplate.VolumeMounts = append(ts.StepTemplate.VolumeMounts, volumeMount)
		for si := range ts.Sidecars {
			ts.Sidecars[si].VolumeMounts = append(ts.Sidecars[si].VolumeMounts, volumeMount)
		}
	}

	return &ts, nil
}

func workspaceVolumeMount(vol corev1.Volume, decl v1beta1.WorkspaceDeclaration, bind v1beta1.WorkspaceBinding) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      vol.Name,
		MountPath: decl.GetMountPath(),
		SubPath:   bind.SubPath,
		ReadOnly:  decl.ReadOnly,
	}
}
