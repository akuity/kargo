import { faExclamationCircle } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';

export const WarehouseTooltip = ({ warehouse }: { warehouse?: string }) =>
  warehouse && (
    <Tooltip
      title={
        <>
          Warehouse Override: <br />
          {warehouse}
        </>
      }
      placement='right'
    >
      <FontAwesomeIcon icon={faExclamationCircle} className='ml-2 text-xs text-yellow-500' />
    </Tooltip>
  );
