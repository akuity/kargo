import {
  faBullseye,
  faCircleCheck,
  faQuestionCircle,
  faTimeline,
  faTools,
  faTruckArrowRight
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Select, Tooltip } from 'antd';
import { useContext } from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ColorContext } from '@ui/context/colors';
import { Warehouse } from '@ui/gen/v1alpha1/generated_pb';

import { FreightlineAction } from '../project/pipelines/types';

export const FreightlineHeader = ({
  promotingStage,
  action,
  cancel,
  downstreamSubs,
  selectedWarehouse,
  setSelectedWarehouse,
  warehouses
}: {
  promotingStage?: string;
  action?: FreightlineAction;
  cancel: () => void;
  downstreamSubs?: string[];
  selectedWarehouse: string;
  setSelectedWarehouse: (warehouse: string) => void;
  warehouses: { [key: string]: Warehouse };
}) => {
  const stageColorMap = useContext(ColorContext);
  const { name: projectName } = useParams();

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

  const navigate = useNavigate();

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
          <>
            <div className='flex items-center text-neutral-500 text-xs mr-auto'>
              <FontAwesomeIcon icon={faTimeline} className='mr-2' />
              FREIGHTLINE
            </div>
            <Button
              icon={<FontAwesomeIcon icon={faTools} />}
              className='-mb-1 mr-2'
              onClick={() => {
                navigate(
                  generatePath(paths.warehouse, {
                    name: projectName,
                    warehouseName:
                      selectedWarehouse || warehouses[Object.keys(warehouses)[0]]?.metadata?.name,
                    tab: 'create-freight'
                  })
                );
              }}
            >
              Assemble Freight
            </Button>
            {(Object.keys(warehouses) || []).length > 1 && (
              <div className='mr-4 -mb-1'>
                <Select
                  className='w-48'
                  value={selectedWarehouse}
                  onChange={(value) => setSelectedWarehouse(value)}
                  labelRender={({ label }) => <div className='text-xs font-semibold'>{label}</div>}
                  optionRender={(opt) => (
                    <div className='text-xs font-normal w-full h-full flex items-center'>
                      <div>{opt.label}</div>
                    </div>
                  )}
                  options={[
                    ...(Object.keys(warehouses) || []).map((w) => ({ value: w, label: w })),
                    {
                      value: '',
                      label: 'All Warehouses'
                    }
                  ]}
                />
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};
