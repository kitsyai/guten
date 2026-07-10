export type TemplateLibrary = "gutenkit";

const NPM_CDN_BASE = "https://cdn.jsdelivr.net/npm";
const GH_CDN_BASE = "https://cdn.jsdelivr.net/gh";
const PACKAGE_BY_LIBRARY: Record<TemplateLibrary, string> = {
  gutenkit: "@kitsy/gutenkit",
};
const REGISTRY_PATH_BY_LIBRARY: Record<TemplateLibrary, string> = {
  gutenkit: "templates",
};
const GITHUB_REPO_BY_LIBRARY: Record<TemplateLibrary, string> = {
  gutenkit: "kitsyai/gutenkit",
};

export interface TemplateRegistryEntry {
  name: string;
  kind: string;
  path: string;
  description: string;
}

export interface TemplateRegistry {
  version: number;
  templates: TemplateRegistryEntry[];
}

export interface TemplateRegistryOptions {
  version?: string;
  host?: string;
  packageName?: string;
  fetchImpl?: (url: string, init?: RequestInit) => Promise<Response>;
}

export interface TemplateRegistryFallbackOptions {
  version?: string;
  host?: string;
  githubHost?: string;
  githubRepo?: string;
  githubTag?: string;
  packageName?: string;
  fetchImpl?: (url: string, init?: RequestInit) => Promise<Response>;
}

function buildNpmRegistryUrl(
  packageName: string,
  version: string,
  host: string,
  registryPath: string,
): string {
  return `${host}/${packageName}@${version}/${registryPath}/index.json`;
}

function normalizeGitTag(version: string): string {
  if (version === "latest" || version === "main") {
    return "main";
  }
  if (version.startsWith("v")) {
    return version;
  }
  return `v${version}`;
}

function buildGithubRegistryUrl(
  version: string,
  host: string,
  repo: string,
): string {
  const tag = normalizeGitTag(version);
  return `${host}/${repo}@${tag}/templates/index.json`;
}

/** Public URL for the template registry index served from npm CDN. */
export function templateRegistryUrl(
  library: TemplateLibrary = "gutenkit",
  options: TemplateRegistryOptions = {},
): string {
  const {
    version = "latest",
    host = NPM_CDN_BASE,
    packageName = PACKAGE_BY_LIBRARY[library],
  } = options;
  return buildNpmRegistryUrl(packageName, version, host, REGISTRY_PATH_BY_LIBRARY[library]);
}

/**
 * Fallback URL for the same registry from github CDN.
 */
export function templateRegistryFallbackUrl(
  library: TemplateLibrary = "gutenkit",
  options: TemplateRegistryFallbackOptions = {},
): string {
  const {
    version = "latest",
    githubHost = GH_CDN_BASE,
    githubRepo = GITHUB_REPO_BY_LIBRARY[library],
    githubTag,
  } = options;
  const tag = githubTag ?? normalizeGitTag(version);
  return buildGithubRegistryUrl(tag, githubHost, githubRepo);
}

/** Returns primary and fallback registry URLs (npm + github). */
export function templateRegistryUrls(
  library: TemplateLibrary = "gutenkit",
  options: TemplateRegistryFallbackOptions = {},
): { npm: string; github: string } {
  return {
    npm: templateRegistryUrl(library, options),
    github: templateRegistryFallbackUrl(library, options),
  };
}

/**
 * Loads the public template registry from npm CDN, and falls back to GitHub CDN
 * for `gutenkit` if npm is missing.
 */
export async function fetchTemplateRegistry(
  library: TemplateLibrary = "gutenkit",
  options: TemplateRegistryFallbackOptions = {},
): Promise<TemplateRegistry> {
  const {
    version = "latest",
    host = NPM_CDN_BASE,
    packageName = PACKAGE_BY_LIBRARY[library],
    fetchImpl = globalThis.fetch,
  } = options;

  const primary = buildNpmRegistryUrl(packageName, version, host, REGISTRY_PATH_BY_LIBRARY[library]);
  const primaryRes = await fetchImpl(primary);
  if (primaryRes.ok) {
    return (await primaryRes.json()) as TemplateRegistry;
  }

  if (library === "gutenkit" && primaryRes.status === 404) {
    const fallback = templateRegistryFallbackUrl(library, options);
    const fallbackRes = await fetchImpl(fallback);
    if (fallbackRes.ok) {
      return (await fallbackRes.json()) as TemplateRegistry;
    }
    throw new Error(
      `guten: failed to fetch template registry ${library}. npm: ${primaryRes.status} ${primaryRes.statusText}; github: ${fallbackRes.status} ${fallbackRes.statusText}`,
    );
  }

  throw new Error(`guten: failed to fetch template registry ${library}: ${primaryRes.status} ${primaryRes.statusText}`);
}

/**
 * Computes a jsDelivr URL for a specific template asset (template.json, sample.json,
 * theme.json, or a liquid part file) from npm package CDN.
 */
export function templateAssetUrl(
  library: TemplateLibrary = "gutenkit",
  assetPath: string,
  options: TemplateRegistryOptions = {},
): string {
  const { version = "latest", host = NPM_CDN_BASE, packageName = PACKAGE_BY_LIBRARY[library] } = options;
  const prefix = REGISTRY_PATH_BY_LIBRARY[library];
  let cleaned = assetPath.replace(/^\/+/, "");
  if (cleaned.startsWith(`${prefix}/`)) {
    cleaned = cleaned.slice(prefix.length + 1);
  }
  cleaned = cleaned.replace(/^templates?\//, "");
  return `${host}/${packageName}@${version}/${REGISTRY_PATH_BY_LIBRARY[library]}/${cleaned}`;
}

/** Computes the github CDN fallback for the same asset. */
export function templateAssetFallbackUrl(
  library: TemplateLibrary = "gutenkit",
  assetPath: string,
  options: TemplateRegistryFallbackOptions = {},
): string {
  const {
    version = "latest",
    githubHost = GH_CDN_BASE,
    githubRepo = GITHUB_REPO_BY_LIBRARY[library],
    githubTag,
  } = options;
  const prefix = REGISTRY_PATH_BY_LIBRARY[library];
  let cleaned = assetPath.replace(/^\/+/, "");
  if (cleaned.startsWith(`${prefix}/`)) {
    cleaned = cleaned.slice(prefix.length + 1);
  }
  cleaned = cleaned.replace(/^templates?\//, "");
  const tag = githubTag ?? normalizeGitTag(version);
  return `${githubHost}/${githubRepo}@${tag}/${prefix}/${cleaned}`;
}

/** Returns primary and fallback asset URLs (npm + github). */
export function templateAssetUrls(
  library: TemplateLibrary = "gutenkit",
  assetPath: string,
  options: TemplateRegistryFallbackOptions = {},
): { npm: string; github: string } {
  return {
    npm: templateAssetUrl(library, assetPath, options),
    github: templateAssetFallbackUrl(library, assetPath, options),
  };
}

export const defaultTemplateLibrary = "gutenkit";
export const defaultTemplateVersion = "latest";
