import { faArrowsLeftRightToLine } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';

import { FreightMode } from '../project/pipelines/types';

import { FreightItem } from './freight-item';

export const FreightSeparator = ({
  count,
  onClick,
  oldest
}: {
  count: number;
  onClick: () => void;
  oldest?: boolean;
}) => {
  return (
    <FreightItem
      onClick={onClick}
      empty={true}
      highlighted={false}
      key='collapsed'
      mode={FreightMode.Default}
      onHover={() => null}
      hideLabel={true}
      childClassname='flex flex-col text-gray-300 text-sm text-center'
    >
      <FontAwesomeIcon icon={faArrowsLeftRightToLine} className='text-gray-300 mb-2' size='2x' />
      {count} {oldest ? 'old' : 'hidden'} freight
    </FreightItem>
  );
};
