import { IconDefinition } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';

import { ArtifactMenuItems } from './artifact-menu-item';
import { DiscoveryResult } from './types';

export const ArtifactMenuGroup = <T extends DiscoveryResult>({
  label,
  icon,
  items,
  onClick,
  selected,
  getKey
}: {
  label: string;
  icon: IconDefinition | null;
  items: T[];
  onClick: (item: T) => void;
  selected?: T;
  getKey: (item: T) => string;
}) =>
  items?.length > 0 && (
    <div className='mb-5'>
      <div className='flex items-center text-gray-400 font-medium uppercase text-xs mb-2'>
        {icon && <FontAwesomeIcon icon={icon} className='mr-2' />}
        <span>{label}</span>
      </div>
      <div>
        <ArtifactMenuItems onClick={onClick} selected={selected} items={items} getKey={getKey} />
      </div>
    </div>
  );
