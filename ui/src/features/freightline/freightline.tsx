import React from 'react';

import { Stage } from '@ui/gen/v1alpha1/generated_pb';

export const Freightline = ({
  children
}: {
  promotingStage?: Stage;
  setPromotingStage: (stage?: Stage) => void;
  children: React.ReactNode;
}) => {
  return (
    <div className='w-full py-3 flex flex-col overflow-hidden' style={{ backgroundColor: '#222' }}>
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
