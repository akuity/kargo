export const shortVersion = (version: string = '', length = 12) => {
  if (version.length <= length) {
    return version;
  }

  const prefix = version.slice(0, length / 2);
  const suffix = version.slice(-(length / 2));

  return `${prefix}...${suffix}`;
};
