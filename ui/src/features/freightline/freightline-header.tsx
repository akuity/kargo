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

import { FreightlineAction } from '../project/pipelines/types';

export const FreightlineHeader = ({
  promotingStage,
  action,
  cancel,
  downstreamSubs
}: {
  promotingStage?: string;
  action?: FreightlineAction;
  cancel: () => void;
  downstreamSubs?: string[];
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
    <div className='w-full pl-6 h-8 mb-3 flex flex-col justify-end font-semibold text-sm'>
      <div className='flex items-end'>
        {action ? (
          <>
            <div className='flex items-center'>
              <FontAwesomeIcon icon={getIcon(action)} className='mr-2' />
              {promotingStage && action != 'manualApproval' ? (
                <>
                  PROMOTING{' '}
                  {action === 'promoteSubscribers'
                    ? `TO ${(downstreamSubs || []).length} DOWNSTREAM SUBSCRIBERS (${downstreamSubs?.join(', ')}) OF`
                    : ''}{' '}
                  STAGE :{' '}
                  <div
                    className='px-2 rounded text-white ml-2 font-semibold'
                    style={{
                      backgroundColor: stageColorMap[promotingStage]
                    }}
                  >
                    {promotingStage.toUpperCase()}
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
            </div>

            <div
              className='ml-auto mr-4 cursor-pointer px-2 py-1 text-white bg-zinc-700 rounded hover:bg-zinc-600 font-semibold text-sm'
              onClick={cancel}
            >
              CANCEL
            </div>
          </>
        ) : (
          <div className='flex items-center text-neutral-500 text-xs'>
            <FontAwesomeIcon icon={faTimeline} className='mr-2' />
            FREIGHTLINE
          </div>
        )}
      </div>
    </div>
  );
};
