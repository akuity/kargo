import { faBan } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Descriptions, Tag, Tooltip } from 'antd';
import { format } from 'date-fns';

import type { Freight } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';
import type { PlainMessageRecursive } from '@ui/utils/connectrpc-utils';

import { isFreightRejected } from './rejection-utils';

export const RejectedFreightTag = ({
  freight,
  compact
}: {
  freight?: Freight | PlainMessageRecursive<Freight>;
  compact?: boolean;
}) => {
  if (!isFreightRejected(freight)) {
    return null;
  }

  return (
    <Tooltip title='Rejected Freight cannot be promoted.'>
      <Tag color='error' className={compact ? 'm-0' : undefined}>
        <FontAwesomeIcon icon={faBan} className={compact ? undefined : 'mr-1'} />
        {!compact && 'Rejected'}
      </Tag>
    </Tooltip>
  );
};

export const RejectedFreightDetails = ({ freight }: { freight?: Freight }) => {
  const rejection = freight?.status?.rejected;
  if (!rejection) {
    return null;
  }

  const rejectedAt = rejection.rejectedAt ? timestampDate(rejection.rejectedAt) : undefined;

  return (
    <Descriptions
      className='mb-5'
      column={1}
      size='small'
      bordered
      title={<RejectedFreightTag freight={freight} />}
      items={[
        {
          label: 'rejected at',
          children: rejectedAt ? format(rejectedAt, 'MMM do yyyy HH:mm:ss') : ''
        },
        {
          label: 'actor',
          children: rejection.actor || ''
        },
        {
          label: 'reason',
          children: rejection.reason || ''
        }
      ]}
    />
  );
};
