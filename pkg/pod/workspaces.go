package pod

import (
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/workspace"
)

type pathName = string
type workspaceName = string
type pathMap = map[pathName]string
type workspacePathMap = map[workspaceName]pathMap

// TODO: need a better type name than this.
type pathPair struct {
	decl v1beta1.WorkspacePath
	bind v1beta1.WorkspacePath
}

func expectedPaths(taskSpec *v1beta1.TaskSpec, taskRunSpec *v1beta1.TaskRunSpec) workspacePathMap {
	paths := workspacePathMap{}

	for _, ws := range taskSpec.Workspaces {
		allPaths := map[string]pathPair{}

		paths[ws.Name] = pathMap{}

		var binding *v1beta1.WorkspaceBinding = nil
		for _, trws := range taskRunSpec.Workspaces {
			if trws.Name == ws.Name {
				binding = &trws
				break
			}
		}

		if binding != nil {
			for _, p := range ws.Paths.Expected {
				allPaths[p.Name] = pathPair{decl: p}
			}
			for _, p := range binding.Paths.Expected {
				if _, ok := allPaths[p.Name]; !ok {
					allPaths[p.Name] = pathPair{bind: p}
				}
				entry := allPaths[p.Name]
				entry.bind = p
				allPaths[p.Name] = entry
			}
		}

		for pathName, pair := range allPaths {
			paths[ws.Name][pathName] = workspace.GetAbsPath(ws.GetMountPath(), pair.decl, pair.bind)
		}
	}
	return paths
}

func producedPaths(taskSpec *v1beta1.TaskSpec, taskRunSpec *v1beta1.TaskRunSpec) workspacePathMap {
	paths := workspacePathMap{}
	for _, ws := range taskSpec.Workspaces {
		allPaths := map[string]pathPair{}

		paths[ws.Name] = pathMap{}

		var binding *v1beta1.WorkspaceBinding = nil
		for _, trws := range taskRunSpec.Workspaces {
			if trws.Name == ws.Name {
				binding = &trws
				break
			}
		}

		if binding != nil {
			for _, p := range ws.Paths.Produced {
				allPaths[p.Name] = pathPair{decl: p}
			}
			for _, p := range binding.Paths.Produced {
				if _, ok := allPaths[p.Name]; !ok {
					allPaths[p.Name] = pathPair{bind: p}
				}
				entry := allPaths[p.Name]
				entry.bind = p
				allPaths[p.Name] = entry
			}
		}

		for pathName, pair := range allPaths {
			paths[ws.Name][pathName] = workspace.GetAbsPath(ws.GetMountPath(), pair.decl, pair.bind)
		}
	}
	return paths
}
