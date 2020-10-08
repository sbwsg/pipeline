package workspace

import (
	"path/filepath"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// FilesToMap takes a list of workspace file entry and converts it into a JSON-ready format
// of {"file-name": "/absolute/path/to/file"}.
func FilesToMap(pathPrefix string, declaredFiles []v1beta1.WorkspaceFileEntry, boundFiles []v1beta1.WorkspaceFileEntry) map[string]string {
	fileMap := map[string]string{}
	for _, df := range declaredFiles {
		var bf v1beta1.WorkspaceFileEntry
		for _, boundFile := range boundFiles {
			if boundFile.Name == df.Name {
				bf = boundFile
				break
			}
		}
		fileMap[df.Name] = filepath.Join(pathPrefix, workspaceFilePath(df, bf))
	}
	return fileMap
}

func workspaceFilePath(declaredFile v1beta1.WorkspaceFileEntry, boundFile v1beta1.WorkspaceFileEntry) string {
	if boundFile.Path != "" {
		return boundFile.Path
	}
	if declaredFile.Path != "" {
		return declaredFile.Path
	}
	return declaredFile.Name
}
