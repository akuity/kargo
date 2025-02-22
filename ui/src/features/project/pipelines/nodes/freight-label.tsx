import { faDocker, faGitAlt } from '@fortawesome/free-brands-svg-icons';
import { faAnchor } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import { format, formatDistance } from 'date-fns';
import { useEffect, useState } from 'react';

import { CommitInfo } from '@ui/features/common/commit-info';
import { Freight } from '@ui/gen/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { getAlias } from '../../../common/utils';

export const FreightLabel = ({ freight }: { freight?: Freight }) => {
  const [copied, setCopied] = useState<boolean>(false);

  useEffect(() => {
    if (copied) {
      const timeout = setTimeout(() => setCopied(false), 1000);
      return () => clearTimeout(timeout);
    }
  }, [copied]);

  const id = freight?.metadata?.name?.substring(0, 7);
  const alias = getAlias(freight);

  const humanReadable = formatDistance(
    timestampDate(freight?.metadata?.creationTimestamp) || 0,
    new Date(),
    {
      addSuffix: true
    }
  );

  return (
    <div
      className='cursor-pointer font-semibold min-w-0 w-full text-center'
      onClick={(e) => {
        if (alias || id) {
          e.preventDefault();
          e.stopPropagation();
          navigator.clipboard.writeText(alias || id || '');
          setCopied(true);
        }
      }}
    >
      {alias || id ? (
        <>
          <Tooltip
            className='w-full font-mono'
            style={{ padding: '0 3px' }}
            title={`ID: ${id}`}
            placement='right'
          >
            <div className='truncate'>{alias || id}</div>
          </Tooltip>
          {freight?.metadata?.creationTimestamp && (
            <Tooltip
              title={format(
                timestampDate(freight?.metadata?.creationTimestamp) || '',
                'MMM do yyyy HH:mm:ss'
              )}
              className='text-[9px] text-gray-400'
              placement='right'
            >
              Created {humanReadable}
            </Tooltip>
          )}
          <div className='flex items-center gap-1 my-1 justify-center text-gray-500'>
            {(freight?.images || []).map((image) => {
              const key = `${image.repoURL}:${image?.tag}`;
              return (
                <Tooltip key={key} title={key} placement='bottom'>
                  <FontAwesomeIcon icon={faDocker} />
                </Tooltip>
              );
            })}
            {(freight?.commits || []).map((commit) => (
              <Tooltip key={commit.id} title={<CommitInfo commit={commit} />} placement='bottom'>
                <FontAwesomeIcon icon={faGitAlt} />
              </Tooltip>
            ))}
            {(freight?.charts || []).map((chart) => {
              const key = chart.repoURL;
              return (
                <Tooltip key={key} title={key} placement='bottom'>
                  <FontAwesomeIcon icon={faAnchor} />
                </Tooltip>
              );
            })}
          </div>
        </>
      ) : (
        <div className='flex items-center justify-center text-gray-400'>NO FREIGHT</div>
      )}
    </div>
  );
};
