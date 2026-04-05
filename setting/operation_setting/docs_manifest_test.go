package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateDocsManifestPath_AllowsEnterpriseStaticManifest(t *testing.T) {
	require.NoError(t, ValidateDocsManifestPath("/enterprise-docs/apifox/manifest.json"))
}

func TestValidateDocsManifestPath_AllowsEmptyValue(t *testing.T) {
	require.NoError(t, ValidateDocsManifestPath(""))
}

func TestValidateDocsManifestPath_RejectsUnsafePaths(t *testing.T) {
	require.Error(t, ValidateDocsManifestPath("https://evil.com/manifest.json"))
	require.Error(t, ValidateDocsManifestPath("/docs/manifest.json"))
	require.Error(t, ValidateDocsManifestPath("/enterprise-docs/apifox/index.json"))
	require.Error(t, ValidateDocsManifestPath("/enterprise-docs/../secret/manifest.json"))
	require.Error(t, ValidateDocsManifestPath(`/enterprise-docs\apifox\manifest.json`))
}
