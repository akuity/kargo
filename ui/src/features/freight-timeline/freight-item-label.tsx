import { faCheck, faClipboard } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import { format, formatDistance } from 'date-fns';
import { useEffect, useState } from 'react';

import { Freight } from '@ui/gen/v1alpha1/generated_pb';
import { timestampDate, PlainMessageRecursive } from '@ui/utils/connectrpc-utils';

import { getAlias } from '../common/utils';

export const FreightItemLabel = ({ freight }: { freight?: PlainMessageRecursive<Freight> }) => {
  const [copied, setCopied] = useState<boolean>(false);

  useEffect(() => {
    if (copied) {
      const timeout = setTimeout(() => setCopied(false), 1000);
      return () => clearTimeout(timeout);
    }
  }, [copied]);

  const id = freight?.metadata?.name?.substring(0, 7);
  const alias = getAlias(freight);
  const aliasLabel =
    Number(alias?.length || 0) > 9 // 9 chars is the max length which will fit on one line
      ? alias?.split('-').map((s, i) => (
          <div className='truncate' key={i}>
            {s}
            {i === 0 && '-'}
          </div>
        ))
      : alias;

  const humanReadable = formatDistance(
    timestampDate(freight?.metadata?.creationTimestamp) || 0,
    new Date(),
    {
      addSuffix: true
    }
  );

  const freightCreationTimestamp = freight?.metadata?.creationTimestamp
    ? timestampDate(freight?.metadata?.creationTimestamp)
    : null;

  return (
    (alias || id) && (
      <div
        className='cursor-pointer font-mono font-semibold min-w-0 w-full'
        onClick={(e) => {
          if (alias || id) {
            e.preventDefault();
            e.stopPropagation();
            navigator.clipboard.writeText(alias || id || '');
            setCopied(true);
          }
        }}
      >
        <Tooltip
          overlayStyle={{ maxWidth: '320px' }}
          placement='right'
          title={
            <>
              <div className='uppercase text-xs w-full text-center font-semibold text-gray-400'>
                <FontAwesomeIcon icon={copied ? faCheck : faClipboard} className='mr-2' />
                {copied ? 'Copied' : `Click to copy ${alias ? 'alias' : 'id'}`}
              </div>
              {alias && <Info title='Alias'>{alias}</Info>}
              <Info title='ID'>
                <div className='font-mono'>{id}</div>
              </Info>
              {freightCreationTimestamp && (
                <Info title='Created'>
                  <div className='text-right'>
                    {format(freightCreationTimestamp, 'MMM do yyyy HH:mm:ss')}
                    <br />({humanReadable})
                  </div>
                </Info>
              )}
            </>
          }
        >
          <div
            className={'hover:text-gray-600 w-full h-8 flex justify-center items-end'}
            style={{ padding: '0 3px' }}
          >
            <div className='truncate'>{aliasLabel || id}</div>
          </div>
        </Tooltip>
      </div>
    )
  );
};

const Info = ({ title, children }: { title: string; children: React.ReactNode }) => (
  <div className='flex items-center my-1'>
    <div className='text-xs text-gray-400 mr-4'>{title}</div>
    <div className='text-sm ml-auto'>{children}</div>
  </div>
);
