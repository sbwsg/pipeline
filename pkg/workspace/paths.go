package workspace

import (
	"path/filepath"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func PathReplacements(mountPath string, declaredPaths []v1beta1.WorkspacePath, boundPaths []v1beta1.WorkspacePath) map[string]string {
	pathReplacements := map[string]string{}

	declaredPathsMap := map[string]v1beta1.WorkspacePath{}
	boundPathsMap := map[string]v1beta1.WorkspacePath{}

	for _, p := range declaredPaths {
		declaredPathsMap[p.Name] = p
	}
	for _, p := range boundPaths {
		boundPathsMap[p.Name] = p
	}

	for name, p := range declaredPathsMap {
		pathReplacements[p.Name] = GetAbsPath(mountPath, p, boundPathsMap[name])
	}

	for name, p := range boundPathsMap {
		pathReplacements[p.Name] = GetAbsPath(mountPath, declaredPathsMap[name], p)
	}

	return pathReplacements
}

func GetAbsPath(mountPath string, declaredPath v1beta1.WorkspacePath, boundPath v1beta1.WorkspacePath) string {
	if boundPath.Path != "" {
		return filepath.Join(mountPath, boundPath.Path)
	}
	if declaredPath.Path != "" {
		return filepath.Join(mountPath, declaredPath.Path)
	}
	if boundPath.Name != "" {
		return filepath.Join(mountPath, boundPath.Name)
	}
	return filepath.Join(mountPath, declaredPath.Name)
}
