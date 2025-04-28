export const shortVersion = (version: string = '') => {
  if (version.length <= 10) {
    return version;
  }

  const prefix = version.slice(0, 7);
  const suffix = version.slice(-6);

  return `${prefix}...${suffix}`;
};
