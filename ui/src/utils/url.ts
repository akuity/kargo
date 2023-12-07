export const urlWithProtocol = (image: string): string | undefined => {
  if (image.startsWith('quay.io') || image.startsWith('docker.io') || image.startsWith('ghcr.io')) {
    return `https://${image}`;
  } else if (image.includes('/')) {
    return `https://hub.docker.com/r/${image}`;
  } else {
    return `https://hub.docker.com/_/${image}`;
  }
};
