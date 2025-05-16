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

  return getGitCommitURL(url, revision);
};

export const getGitCommitURL = (url: string, revision: string) => {
  let baseUrl;

  if (url.includes('github.com')) {
    baseUrl = url
      .replace(/^git@github.com:/, 'https://github.com/')
      .replace(/^https?:\/\/github.com\//, 'https://github.com/')
      .replace(/\.git$/, '');
    return `${baseUrl}/commit/${revision}`;
  } else if (url.includes('gitlab.com')) {
    baseUrl = url
      .replace(/^git@gitlab.com:/, 'https://gitlab.com/')
      .replace(/^https?:\/\/gitlab.com\//, 'https://gitlab.com/')
      .replace(/\.git$/, '');
    return `${baseUrl}/-/commit/${revision}`;
  } else if (url.includes('bitbucket.org')) {
    baseUrl = url
      .replace(/^git@bitbucket.org:/, 'https://bitbucket.org/')
      .replace(/^https?:\/\/bitbucket.org\//, 'https://bitbucket.org/')
      .replace(/\.git$/, '');
    return `${baseUrl}/commits/${revision}`;
  }

  return url;
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
