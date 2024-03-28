import {
  faBullseye,
  faCircleCheck,
  faQuestionCircle,
  faTimeline,
  faTruckArrowRight
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import { useContext } from 'react';

import { ColorContext } from '@ui/context/colors';
import { Stage } from '@ui/gen/v1alpha1/generated_pb';

import { FreightlineAction } from '../project/project-details/types';

export const FreightlineHeader = ({
  promotingStage,
  action,
  cancel
}: {
  promotingStage?: Stage;
  action?: FreightlineAction;
  cancel: () => void;
}) => {
  const stageColorMap = useContext(ColorContext);

  const getIcon = (action: FreightlineAction) => {
    switch (action) {
      case 'promote':
        return faBullseye;
      case 'promoteSubscribers':
        return faTruckArrowRight;
      case 'manualApproval':
        return faCircleCheck;
      default:
        return faQuestionCircle;
    }
  };

  return (
    <div className='w-full p-1 pl-6 h-12 mb-3 flex flex-col justify-end font-semibold text-sm'>
      <div className='flex items-center'>
        {action ? (
          <>
            <FontAwesomeIcon icon={getIcon(action)} className='mr-2' />
            {promotingStage && action != 'manualApproval' ? (
              <>
                PROMOTING {action === 'promoteSubscribers' ? 'SUBSCRIBERS OF' : ''} STAGE:{' '}
                <div
                  className='px-2 py-1 rounded text-white ml-2 font-semibold'
                  style={{
                    backgroundColor: stageColorMap[promotingStage?.metadata?.uid || '']
                  }}
                >
                  {promotingStage?.metadata?.name?.toUpperCase()}
                </div>
                <Tooltip
                  title={
                    <>
                      Available freight are any which have been verified in{' '}
                      {action === 'promote' && 'any immediately upstream stage OR approved for'}{' '}
                      this stage.
                    </>
                  }
                >
                  <FontAwesomeIcon
                    icon={faQuestionCircle}
                    className='cursor-pointer text-zinc-500 ml-2'
                  />
                </Tooltip>
              </>
            ) : (
              <>MANUALLY APPROVING FREIGHT</>
            )}

            <div
              className='ml-auto mr-4 cursor-pointer px-2 py-1 text-white bg-zinc-700 rounded hover:bg-zinc-600 font-semibold text-sm'
              onClick={cancel}
            >
              CANCEL
            </div>
          </>
        ) : (
          <>
            <FontAwesomeIcon icon={faTimeline} className='mr-2' />
            FREIGHTLINE
          </>
        )}
      </div>
    </div>
  );
};
