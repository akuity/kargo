import { faTowerBroadcast } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import { formatDistance } from 'date-fns';

import { ShardInfo, ShardStatus } from '@ui/gen/api/v2/models';

const ALIVE_COLOR = '#52c41a';
const DEAD_COLOR = '#f5222d';

export const ShardStatusIcon = (props: { shard?: ShardInfo; shardName?: string }) => {
  const { shard, shardName } = props;

  if (!shard) {
    return null;
  }

  const isAlive = shard.status === ShardStatus.shardStatusAlive;
  const color = isAlive ? ALIVE_COLOR : DEAD_COLOR;
  const name = shardName || shard.name || '';

  const lastSeen = shard.lastSeen
    ? formatDistance(new Date(shard.lastSeen), new Date(), { addSuffix: true })
    : 'never';

  return (
    <Tooltip
      title={
        <div className='text-xs'>
          <div>
            <b>Shard:</b> {name}
          </div>
          <div>
            <b>Agent:</b> {isAlive ? 'alive' : 'dead'}
          </div>
          <div>
            <b>Last heartbeat:</b> {lastSeen}
          </div>
        </div>
      }
    >
      <span
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          justifyContent: 'center',
          width: 22,
          height: 22,
          borderRadius: '50%',
          background: '#fff',
          border: '1px solid rgba(0, 0, 0, 0.1)',
          boxShadow: '0 1px 1px rgba(0, 0, 0, 0.08)'
        }}
      >
        <FontAwesomeIcon icon={faTowerBroadcast} className='text-[10px]' style={{ color }} />
      </span>
    </Tooltip>
  );
};
