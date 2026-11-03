import { faExpand } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Tooltip } from 'antd';

// Opens a step's Logs, Output or Config in a large modal. Shared so that it
// looks and sits the same at the end of every panel's toolbar.
export const FullViewButton = ({ onClick }: { onClick(): void }) => (
  <Tooltip title='Open in a large view'>
    <Button
      size='small'
      type='text'
      icon={<FontAwesomeIcon icon={faExpand} className='text-xs' />}
      onClick={onClick}
    >
      Full view
    </Button>
  </Tooltip>
);
