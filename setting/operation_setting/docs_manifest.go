package operation_setting

import (
	"fmt"
	"path"
	"strings"
)

func ValidateDocsManifestPath(raw string) error {
	manifestPath := strings.TrimSpace(raw)
	if manifestPath == "" {
		return nil
	}

	if !strings.HasPrefix(manifestPath, "/enterprise-docs/") {
		return fmt.Errorf("docs manifest path must start with /enterprise-docs/")
	}

	if !strings.HasSuffix(manifestPath, "/manifest.json") {
		return fmt.Errorf("docs manifest path must end with /manifest.json")
	}

	if path.Clean(manifestPath) != manifestPath || strings.Contains(manifestPath, `\`) {
		return fmt.Errorf("docs manifest path must be a normalized static path")
	}

	return nil
}
