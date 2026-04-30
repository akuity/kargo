import { IconDefinition } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';

export const SegmentLabel = ({
  icon,
  children
}: {
  icon?: IconDefinition;
  children: React.ReactNode;
}) => (
  <span className='flex items-center font-semibold justify-center text-center p-4'>
    {icon && <FontAwesomeIcon icon={icon} className='mr-2' />}
    {children}
  </span>
);
