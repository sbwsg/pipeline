package workspace

import (
	"path/filepath"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// PathsToMap takes a list of workspace path entry and converts it into a JSON-ready format
// of {"path-name": "/absolute/path"}.
func PathsToMap(pathPrefix string, declaredPaths []v1beta1.WorkspacePathEntry, boundPaths []v1beta1.WorkspacePathEntry) map[string]string {
	pathMap := map[string]string{}
	for _, dp := range declaredPaths {
		var bp v1beta1.WorkspacePathEntry
		for _, boundPath := range boundPaths {
			if boundPath.Name == dp.Name {
				bp = boundPath
				break
			}
		}
		pathMap[dp.Name] = filepath.Join(pathPrefix, workspacePath(dp, bp))
	}
	return pathMap
}

func workspacePath(declaredPath v1beta1.WorkspacePathEntry, boundPath v1beta1.WorkspacePathEntry) string {
	if boundPath.Path != "" {
		return boundPath.Path
	}
	if declaredPath.Path != "" {
		return declaredPath.Path
	}
	return declaredPath.Name
}
