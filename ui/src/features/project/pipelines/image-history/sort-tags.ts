import semver from 'semver';

export const sortTags = (tags: string[]): string[] => {
  return tags.sort((a, b) => {
    try {
      return semver.compare(b, a);
    } catch {
      return b.localeCompare(a);
    }
  });
};
