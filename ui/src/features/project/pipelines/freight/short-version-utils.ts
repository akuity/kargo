export const shortVersion = (version: string = '') => {
  if (version.length <= 12) {
    return version;
  }

  const prefix = version.slice(0, 6);
  const suffix = version.slice(-6);

  return `${prefix}...${suffix}`;
};
