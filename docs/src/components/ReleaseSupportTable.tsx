import React, {useEffect, useState} from 'react';
import useIsBrowser from '@docusaurus/useIsBrowser';
import {
  bestReleasesURL,
  coverageEnd,
  fallbackReleases,
  formatDate,
  parseBestReleases,
  parseReleaseDate,
  relativePhrase,
  statusLabels,
  statusOf,
  type Release,
} from './releaseSupport';
import styles from './ReleaseSupportTable.module.css';

export default function ReleaseSupportTable(): JSX.Element {
  // The absolute dates below are facts and render identically everywhere, but
  // anything derived from the current time would differ between the build and
  // the visitor's clock. useIsBrowser is false during SSR and during the first
  // hydration render, so those cells stay empty until React is live and then
  // fill in without a hydration mismatch.
  const isBrowser = useIsBrowser();
  const now = isBrowser ? new Date() : null;

  // The release workflow republishes best-releases.json on every release, so
  // fetching it keeps the table current between docs builds. Rendering starts
  // from the committed snapshot, which is also what the server rendered, so
  // this refresh never causes a hydration mismatch.
  const [releases, setReleases] = useState<Release[]>(fallbackReleases);
  useEffect(() => {
    const controller = new AbortController();
    fetch(bestReleasesURL, {signal: controller.signal})
      .then((response) =>
        response.ok
          ? response.json()
          : Promise.reject(new Error(`status ${response.status}`)),
      )
      .then((payload) => {
        const fetched = parseBestReleases(payload);
        if (fetched.length > 0) {
          setReleases(fetched);
        }
      })
      .catch(() => {
        // The snapshot is already on screen. A failed refresh is not something
        // the reader can act on, so it is not worth surfacing.
      });
    return () => controller.abort();
  }, []);

  return (
    <table className={styles.table}>
      <thead>
        <tr>
          <th>Release</th>
          <th>Released</th>
          <th>Latest patch</th>
          <th>Critical CVE coverage ends</th>
        </tr>
      </thead>
      <tbody>
        {releases.map((release) => {
          const released = parseReleaseDate(release.released);
          const coverageEnds = coverageEnd(released);
          const status = now ? statusOf(coverageEnds, now) : null;
          return (
            <tr key={release.minor}>
              <th scope="row" className={styles.release}>
                {status && (
                  <span
                    className={`${styles.dot} ${styles[status]}`}
                    role="img"
                    aria-label={statusLabels[status]}
                  />
                )}
                {release.minor}
              </th>
              <td>{formatDate(released)}</td>
              <td>
                <code>{release.latestPatch}</code>
              </td>
              <td>
                {formatDate(coverageEnds)}
                {status && (
                  <span className={`${styles.relative} ${styles[status]}`}>
                    {relativePhrase(coverageEnds, now!)}
                  </span>
                )}
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}
