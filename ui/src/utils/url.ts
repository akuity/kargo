// Logic adapted from:
// https://github.com/distribution/reference/blob/v0.5.0/normalize.go

const protocol = 'https://';
const dockerHub = 'hub.docker.com';
const officialPath = '_';
const userPath = 'r';

export const urlForImage = (image: string | undefined | null): string => {
  if (!image) {
    return '';
  }

  const parts = image.split('/');

  if (parts.length > 0 && parts[0] === 'docker.io') {
    parts.shift();
  }

  if (parts.length > 0 && parts[0] === 'library') {
    parts.shift();
  }

  if (parts.length === 0) {
    return '';
  }

  image = parts.join('/');

  if (parts.length === 1) {
    return `${protocol}${dockerHub}/${officialPath}/${image}`;
  }

  if (!parts[0] || typeof parts[0] !== 'string') {
    return `${protocol}${image}`;
  }

  if (
    !(parts[0].includes('.') || parts[0].includes(':')) &&
    parts[0] !== 'localhost' &&
    parts[0].toLowerCase() === parts[0]
  ) {
    return `${protocol}${dockerHub}/${userPath}/${image}`;
  }

  if (parts[0] === 'public.ecr.aws') {
    return `${protocol}gallery.ecr.aws/${parts.slice(1).join('/')}`;
  }

  if (parts[0].endsWith('amazonaws.com')) {
    const domainParts = parts[0].split('.');
    const region = domainParts[3];
    const id = domainParts[0];

    return `${protocol}${region}.console.aws.amazon.com/ecr/repositories/private/${id}/${parts
      .slice(1)
      .join('/')}`;
  }
  return `${protocol}${image}`;
};
