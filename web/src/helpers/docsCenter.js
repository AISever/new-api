export function normalizeDocsManifestPath(rawPath) {
  const manifestPath = typeof rawPath === 'string' ? rawPath.trim() : '';
  if (!manifestPath) {
    return '';
  }

  if (!manifestPath.startsWith('/enterprise-docs/')) {
    return '';
  }

  if (!manifestPath.endsWith('/manifest.json')) {
    return '';
  }

  if (
    manifestPath.includes('..') ||
    manifestPath.includes('\\') ||
    manifestPath.includes('//')
  ) {
    return '';
  }

  return manifestPath;
}

export function getDocsAvailability(status) {
  const docsLink =
    typeof status?.docs_link === 'string' ? status.docs_link.trim() : '';
  const manifestPath = normalizeDocsManifestPath(status?.docs_manifest_path);

  return {
    hasLocalDocs: manifestPath !== '',
    hasLegacyDocsLink: docsLink !== '',
    manifestPath,
    docsLink,
  };
}

export function shouldUseLocalDocs(status) {
  return getDocsAvailability(status).hasLocalDocs;
}
