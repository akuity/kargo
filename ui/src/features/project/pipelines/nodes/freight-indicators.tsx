import { Tooltip } from 'antd';
import classNames from 'classnames';
import { useContext } from 'react';

import { ColorContext } from '@ui/context/colors';
import { getAlias } from '@ui/features/common/utils';
import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

export const FreightIndicators = ({
  freight,
  selectedFreight,
  onClick
}: {
  freight?: Freight[];
  selectedFreight: number;
  onClick: (index: number) => void;
}) => {
  const { warehouseColorMap } = useContext(ColorContext);

  if (!freight || freight.length <= 1) {
    return null;
  }

  return (
    <div className='flex gap-2 justify-center items-center py-1 top-1'>
      {freight.map((freight, idx) => (
        <Tooltip placement='right' title={getAlias(freight)} key={freight?.metadata?.name || idx}>
          <div
            className={classNames('rounded-full mb-2 opacity-50 hover:opacity-30', {
              '!opacity-100': selectedFreight === idx
            })}
            style={{
              width: '10px',
              height: '10px',
              backgroundColor: warehouseColorMap[freight?.origin?.name || '']
            }}
            onClick={(e) => {
              e.stopPropagation();
              onClick(idx);
            }}
          />
        </Tooltip>
      ))}
    </div>
  );
};
