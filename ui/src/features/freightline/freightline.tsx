import React from 'react';

import { Stage } from '@ui/gen/v1alpha1/types_pb';

export type PromotionType = 'default' | 'subscribers';

export const FreightlineHeader = ({
  children,
  banner
}: {
  children?: React.ReactNode;
  banner?: React.ReactNode;
}) => (
  <>
    <div
      className='w-full p-1 pl-12 mb-2 text-xs h-6 flex items-center'
      style={{ backgroundColor: '#111' }}
    >
      {banner}
    </div>

    <div className='flex items-center ml-12'>{children}</div>
  </>
);

export const Freightline = ({
  children,
  header
}: {
  promotingStage?: Stage;
  setPromotingStage: (stage?: Stage) => void;
  promotionType?: PromotionType;
  children: React.ReactNode;
  header?: React.ReactNode;
}) => {
  return (
    <div className='w-full pb-3 flex flex-col overflow-hidden' style={{ backgroundColor: '#222' }}>
      <div className='text-gray-300 text-sm overflow-hidden mb-2'>{header}</div>
      <div className='flex h-44 w-full items-center px-1'>
        <div
          className='text-gray-500 text-sm font-semibold mb-2 w-min h-min'
          style={{ transform: 'rotate(-0.25turn)' }}
        >
          NEW
        </div>
        <div className='flex items-center h-full overflow-x-auto'>{children}</div>
        <div className='rotate-90 text-gray-500 text-sm font-semibold ml-auto'>OLD</div>
      </div>
    </div>
  );
};
