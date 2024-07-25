import { IconDefinition } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';

import { ArtifactMenuItems } from './artifact-menu-item';
import { DiscoveryResult } from './types';

export const ArtifactMenuGroup = ({
  label,
  icon,
  items,
  onClick,
  selected
}: {
  label: string;
  icon: IconDefinition;
  items: DiscoveryResult[];
  onClick: (item: DiscoveryResult) => void;
  selected?: DiscoveryResult;
}) =>
  items?.length > 0 && (
    <div className='mb-5'>
      <div className='flex items-center text-gray-400 font-medium uppercase text-xs mb-2'>
        <FontAwesomeIcon icon={icon} className='mr-2' />
        <span>{label}</span>
      </div>
      <div>
        <ArtifactMenuItems onClick={onClick} selected={selected} items={items} />
      </div>
    </div>
  );
