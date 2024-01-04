import { faCheck, faClipboard } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import { formatDistance } from 'date-fns';
import { useEffect, useState } from 'react';

import { Freight } from '@ui/gen/v1alpha1/types_pb';

const ALIAS_LABEL_KEY = 'kargo.akuity.io/alias';

export const getAlias = (freight?: Freight): string | undefined => {
  return freight?.metadata?.labels[ALIAS_LABEL_KEY] || undefined;
};

export const FreightLabel = ({ freight }: { freight?: Freight }) => {
  const [id, setId] = useState<string>('');
  const [alias, setAlias] = useState<string | undefined>();
  const [copied, setCopied] = useState<boolean>(false);

  useEffect(() => {
    if (copied) {
      const timeout = setTimeout(() => setCopied(false), 1000);
      return () => clearTimeout(timeout);
    }
  }, [copied]);

  useEffect(() => {
    setAlias(freight?.metadata?.labels[ALIAS_LABEL_KEY]);
    setId(freight?.metadata?.name?.substring(0, 7) || 'N/A');
  }, [freight]);

  return (
    <div
      className='truncate cursor-pointer'
      onClick={() => {
        navigator.clipboard.writeText(alias || id);
        setCopied(true);
      }}
    >
      <Tooltip
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
            {freight?.metadata?.creationTimestamp && (
              <Info title='Created'>
                <div className='text-right'>
                  {formatDistance(freight?.metadata?.creationTimestamp?.toDate(), new Date(), {
                    addSuffix: true
                  })}
                </div>
              </Info>
            )}
          </>
        }
      >
        <div className='hover:text-gray-100'>{alias || id}</div>
      </Tooltip>
    </div>
  );
};

const Info = ({ title, children }: { title: string; children: React.ReactNode }) => (
  <div className='flex items-center my-1'>
    <div className='text-xs text-gray-400 mr-4'>{title}</div>
    <div className='text-sm ml-auto'>{children}</div>
  </div>
);
