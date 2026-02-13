import { formatDistance } from 'date-fns';

const ociPrefix = 'org.opencontainers.image';

type Annotation = Record<string, string>;

export const ociAnnotationKeys = {
  // date and time on which the image was built
  createdAt: `${ociPrefix}.created`,
  // URL to get source code for building the image
  source: `${ociPrefix}.source`,
  // Source control revision identifier for the packaged software
  revision: `${ociPrefix}.revision`
};

export const getImageSource = (annotation: Annotation) => {
  const url = annotation?.[ociAnnotationKeys.source];
  const revision = annotation?.[ociAnnotationKeys.revision];

  if (!revision) {
    return url;
  }

  if (!url) {
    return '';
  }

  return getGitCommitURL(url, revision);
};

export const getGitCommitURL = (url: string, revision: string) => {
  if (!url) {
    return '';
  }

  let baseUrl = url;

  if (baseUrl.startsWith('git@')) {
    baseUrl = baseUrl.replace(':', '/').replace(/^git@/, 'https://');
  } else if (!baseUrl.startsWith('http://') && !baseUrl.startsWith('https://')) {
    baseUrl = `https://${baseUrl}`;
  }

  baseUrl = baseUrl.replace(/\.git$/, '');

  try {
    const urlObj = new URL(baseUrl);
    const hostname = urlObj.hostname.toLowerCase();

    if (hostname.includes('github')) {
      return `${baseUrl}/commit/${revision}`;
    }
    if (hostname.includes('gitlab')) {
      return `${baseUrl}/-/commit/${revision}`;
    }
    if (hostname.includes('bitbucket')) {
      return `${baseUrl}/commits/${revision}`;
    }
  } catch {
    // If URL parsing fails, we simply return the baseUrl below
  }

  return baseUrl;
};

export const getImageBuiltDate = (annotation: Annotation) => {
  const buildDate = annotation?.[ociAnnotationKeys.createdAt];

  if (buildDate) {
    return formatDistance(new Date(buildDate), new Date(), { addSuffix: true })?.replace(
      'about',
      ''
    );
  }

  return '';
};

export const splitOciPrefixedAnnotations = (annotation: Annotation) => {
  const ociPrefixedAnnotations: Record<string, string> = {};
  const restAnnotations: Record<string, string> = {};
  for (const [key, value] of Object.entries(annotation)) {
    if (key.startsWith(ociPrefix)) {
      ociPrefixedAnnotations[key] = value;
    } else {
      restAnnotations[key] = value;
    }
  }

  return {
    ociPrefixedAnnotations,
    restAnnotations
  };
};
