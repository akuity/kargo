import { expect, test, describe } from 'vitest';

import { urlForImage } from './url';

const testImages = {
  // Quay
  'quay.io/jitesoft/nginx': 'https://quay.io/jitesoft/nginx',

  // GHCR
  'ghcr.io/jitesoft/nginx': 'https://ghcr.io/jitesoft/nginx',

  // pkg.dev
  'us-docker.pkg.dev/cloudrun/container/hello':
    'https://us-docker.pkg.dev/cloudrun/container/hello',

  // DockerHub Official
  nginx: 'https://hub.docker.com/_/nginx',
  'library/nginx': 'https://hub.docker.com/_/nginx',
  'docker.io/nginx': 'https://hub.docker.com/_/nginx',
  'docker.io/library/nginx': 'https://hub.docker.com/_/nginx',

  // DockerHub User
  'jitesoft/nginx': 'https://hub.docker.com/r/jitesoft/nginx',
  'docker.io/jitesoft/nginx': 'https://hub.docker.com/r/jitesoft/nginx'
} as { [key: string]: string };

describe('urlForImage', () => {
  Object.keys(testImages).forEach((image) => {
    test(`image: ${image}`, () => {
      expect(urlForImage(image)).toBe(testImages[image]);
    });
  });
});
