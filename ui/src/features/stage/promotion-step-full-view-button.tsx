import { faExpand } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button } from 'antd';

// Opens a step's Logs, Output or Config in a large modal.
export const FullViewButton = ({ onClick }: { onClick(): void }) => (
  <Button
    size='small'
    type='text'
    icon={<FontAwesomeIcon icon={faExpand} className='text-xs' />}
    onClick={onClick}
  >
    Full view
  </Button>
);
