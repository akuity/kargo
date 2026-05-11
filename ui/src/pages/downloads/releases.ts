import { useQuery } from '@tanstack/react-query';
import semver from 'semver';

import { Release } from './types';

const BEST_RELEASES_URL = 'https://akuity.github.io/kargo/best-releases.json';

const fetchBestReleases = async (): Promise<Release[]> => {
  const res = await fetch(BEST_RELEASES_URL);
  const data: { releases: Release[] } = await res.json();
  return data.releases ?? [];
};

export const useBestReleases = () =>
  useQuery({
    queryKey: ['best-releases'],
    queryFn: fetchBestReleases
  });

export const majorMinorVersion = (version: string): string => {
  const parsed = semver.parse(version);
  return parsed ? `v${parsed.major}.${parsed.minor}` : version;
};

export const releaseLabel = (release: Release): string => {
  const v = majorMinorVersion(release.version);
  return release.latest ? `${v} (latest)` : v;
};
