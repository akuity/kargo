import { Tooltip } from 'antd';
import { useEffect, useState } from 'react';

import { Freight } from '@ui/gen/v1alpha1/types_pb';

const ALIAS_LABEL_KEY = 'kargo.akuity.com/alias';

export const FreightLabel = ({ freight }: { freight?: Freight }) => {
  const [id, setId] = useState<string>('');
  const [alias, setAlias] = useState<string | undefined>();

  useEffect(() => {
    setAlias(freight?.metadata?.labels[ALIAS_LABEL_KEY]);
    setId(freight?.metadata?.name?.substring(0, 7) || 'N/A');
  }, [freight]);

  return (
    <span className='truncate'>{alias ? <Tooltip title={id}>{alias}</Tooltip> : <>{id}</>}</span>
  );
};
