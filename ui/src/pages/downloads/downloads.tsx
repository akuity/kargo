import { faCodeCommit, faExternalLink } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Select, Spin, Typography } from 'antd';
import { useState } from 'react';

import { PageTitle } from '@ui/features/common';

import { DownloadItem } from './components';
import { GITHUB_RELEASES_URL, PLATFORMS } from './platforms';
import { majorMinorVersion, releaseLabel, useBestReleases } from './releases';

export const Downloads = () => {
  const { data: releases = [], isLoading } = useBestReleases();

  const [selectedVersion, setSelectedVersion] = useState<string | null>(() =>
    __UI_VERSION__ !== 'development' ? majorMinorVersion(__UI_VERSION__) : null
  );

  const effectiveRelease =
    releases.find((r) => !!selectedVersion && majorMinorVersion(r.version) === selectedVersion) ??
    releases.find((r) => r.latest) ??
    releases[0];

  const dropdownOptions = releases.map((r) => ({
    value: majorMinorVersion(r.version),
    label: releaseLabel(r)
  }));

  return (
    <div className='p-6'>
      <PageTitle title='CLI Downloads' />
      <div className='text-2xl mb-2 font-semibold flex items-center gap-2'>
        <FontAwesomeIcon icon={faCodeCommit} className='mr-2' />
        {effectiveRelease ? majorMinorVersion(effectiveRelease.version) : 'Latest version'}
      </div>
      <Typography.Link
        href={GITHUB_RELEASES_URL}
        target='_blank'
        style={{
          display: 'flex',
          alignItems: 'center',
          marginBottom: 16,
          fontSize: 12,
          textTransform: 'uppercase'
        }}
      >
        <FontAwesomeIcon icon={faExternalLink} style={{ marginRight: 8 }} />
        View all releases
      </Typography.Link>
      <div className='mb-6'>
        {isLoading ? (
          <Spin size='small' />
        ) : (
          releases.length > 0 && (
            <Select
              value={majorMinorVersion(effectiveRelease.version)}
              onChange={setSelectedVersion}
              options={dropdownOptions}
              style={{ minWidth: '220px' }}
              placeholder='Select version'
            />
          )
        )}
      </div>
      <div className='flex items-center gap-4 flex-wrap'>
        {PLATFORMS.map((platform) => (
          <DownloadItem key={platform.title} {...platform} release={effectiveRelease} />
        ))}
      </div>
    </div>
  );
};
