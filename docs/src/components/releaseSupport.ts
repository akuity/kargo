// Release support data and the date arithmetic over it. Kept free of React and
// CSS imports so it can be exercised on its own.

export type Release = {
  minor: string;
  // Publication date of the x.y.0 release, as recorded by GitHub. The
  // vulnerability management policy measures backport windows from an
  // affected release's publication date, so that is the date used here.
  released: string;
  latestPatch: string;
};

// A snapshot of the live data below, rendered server-side and used whenever the
// fetch cannot complete -- no JavaScript, an offline reader, a GitHub Pages
// outage. It only needs to be accurate enough to be useful in those cases;
// refresh it with `go run ./hack/best-releases` from the repository root.
export const fallbackReleases: Release[] = [
  {minor: '1.11', released: '2026-07-24', latestPatch: 'v1.11.5'},
  {minor: '1.10', released: '2026-04-17', latestPatch: 'v1.10.13'},
  {minor: '1.9', released: '2026-01-29', latestPatch: 'v1.9.13'},
  {minor: '1.8', released: '2025-10-21', latestPatch: 'v1.8.17'},
  {minor: '1.7', released: '2025-08-05', latestPatch: 'v1.7.10'},
  {minor: '1.6', released: '2025-06-27', latestPatch: 'v1.6.4'},
  {minor: '1.5', released: '2025-05-15', latestPatch: 'v1.5.3'},
  {minor: '1.4', released: '2025-04-05', latestPatch: 'v1.4.4'},
  {minor: '1.3', released: '2025-02-25', latestPatch: 'v1.3.4'},
  {minor: '1.2', released: '2025-01-14', latestPatch: 'v1.2.3'},
  {minor: '1.1', released: '2024-12-06', latestPatch: 'v1.1.3'},
  {minor: '1.0', released: '2024-10-19', latestPatch: 'v1.0.4'},
];

// Published on every release by the publish-best-releases job in
// .github/workflows/release.yaml, so the table refreshes without a docs build.
export const bestReleasesURL =
  'https://akuity.github.io/kargo/best-releases.json';

// Critical CVE fixes are backported to AKP builds of releases published within
// the last 12 months.
export const criticalWindowMonths = 12;

// A release whose coverage ends within this many days is flagged so operators
// have warning before they fall outside the window.
export const endingSoonDays = 90;

const msPerDay = 86_400_000;

// Release dates are parsed as UTC midnight, so every formatter here must also
// read them as UTC or the rendered day can slip by one west of Greenwich.
const dateFormatter = new Intl.DateTimeFormat('en', {
  day: 'numeric',
  month: 'short',
  year: 'numeric',
  timeZone: 'UTC',
});

const relativeFormatter = new Intl.RelativeTimeFormat('en', {numeric: 'auto'});

export type Status = 'supported' | 'endingSoon' | 'ended';

export const statusLabels: Record<Status, string> = {
  supported: 'Covered',
  endingSoon: 'Ending soon',
  ended: 'Ended',
};

export function parseReleaseDate(released: string): Date {
  return new Date(`${released}T00:00:00Z`);
}

export function formatDate(date: Date): string {
  return dateFormatter.format(date);
}

export function coverageEnd(released: Date): Date {
  const result = new Date(released);
  result.setUTCMonth(result.getUTCMonth() + criticalWindowMonths);
  return result;
}

export function statusOf(coverageEnds: Date, now: Date): Status {
  const remainingDays = (coverageEnds.getTime() - now.getTime()) / msPerDay;
  if (remainingDays <= 0) {
    return 'ended';
  }
  if (remainingDays <= endingSoonDays) {
    return 'endingSoon';
  }
  return 'supported';
}

export function relativePhrase(target: Date, now: Date): string {
  const days = Math.round((target.getTime() - now.getTime()) / msPerDay);
  const magnitude = Math.abs(days);
  if (magnitude < 7) {
    return relativeFormatter.format(days, 'day');
  }
  if (magnitude < 45) {
    return relativeFormatter.format(Math.round(days / 7), 'week');
  }
  if (magnitude < 365) {
    return relativeFormatter.format(Math.round(days / 30), 'month');
  }
  return relativeFormatter.format(Math.round(days / 365), 'year');
}

const versionPattern = /^v(\d+)\.(\d+)\.(\d+)$/;
const datePattern = /^\d{4}-\d{2}-\d{2}$/;

type Version = {major: number; minorNumber: number; patch: number};
type ParsedRelease = Release & Version;

function parseVersion(version: string): Version | null {
  const match = versionPattern.exec(version);
  if (!match) {
    return null;
  }
  return {
    major: Number(match[1]),
    minorNumber: Number(match[2]),
    patch: Number(match[3]),
  };
}

// mergeBestReleases overlays a best-releases.json payload onto the releases
// already known -- normally the committed snapshot -- and returns the result
// sorted newest first.
//
// It merges rather than replaces because the published file has not always
// been complete or current: an older copy of the generator truncated it and
// omitted initialReleaseDate. So a fetched entry can only improve the table.
// It updates a known line's latest patch if newer, and adds a line not yet
// known only if it carries the date needed to place it. Lines the payload
// omits keep their snapshot values, and malformed entries are skipped.
export function mergeBestReleases(
  payload: unknown,
  known: Release[],
): Release[] {
  const byMinor = new Map<string, ParsedRelease>();
  for (const release of known) {
    const version = parseVersion(release.latestPatch);
    if (version) {
      byMinor.set(release.minor, {...version, ...release});
    }
  }

  const entries = (payload as {releases?: unknown} | null)?.releases;
  for (const entry of Array.isArray(entries) ? entries : []) {
    const latestPatch = (entry as {version?: unknown})?.version;
    const date = (entry as {initialReleaseDate?: unknown})?.initialReleaseDate;
    if (typeof latestPatch !== 'string') {
      continue;
    }
    const version = parseVersion(latestPatch);
    if (!version) {
      continue;
    }
    const minor = `${version.major}.${version.minorNumber}`;
    const existing = byMinor.get(minor);
    const released =
      typeof date === 'string' && datePattern.test(date)
        ? date
        : existing?.released;
    if (!released) {
      continue;
    }
    if (existing && existing.patch > version.patch) {
      continue;
    }
    byMinor.set(minor, {...version, minor, released, latestPatch});
  }

  return [...byMinor.values()]
    .sort((a, b) => b.major - a.major || b.minorNumber - a.minorNumber)
    .map(({minor, released, latestPatch}) => ({minor, released, latestPatch}));
}
