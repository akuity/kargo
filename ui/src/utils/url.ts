// Logic adapted from:
// https://github.com/distribution/distribution/blob/7b502560cad43970472964166dcb095b1f883ae4/reference/normalize.go

const protocol = 'https://';
const dockerHub = 'hub.docker.com';
const officialPath = '_';
const userPath = 'r';

export const urlForImage = (image: string): string => {
  const parts = image.split('/');

  if (parts[0] === 'docker.io') {
    parts.shift();
  }

  if (parts[0] === 'library') {
    parts.shift();
  }

  image = parts.join('/');

  if (parts.length === 1) {
    return `${protocol}${dockerHub}/${officialPath}/${image}`;
  } else if (
    !(parts[0].includes('.') || parts[0].includes(':')) &&
    parts[0] !== 'localhost' &&
    parts[0].toLowerCase() === parts[0]
  ) {
    return `${protocol}${dockerHub}/${userPath}/${image}`;
  }
  return `${protocol}${image}`;
};
