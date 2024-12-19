import {
  faBuilding,
  faBullseye,
  faCalendarPlus,
  faCalendarXmark,
  faCircleCheck,
  faCompress,
  faExpand,
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

import { CollapseMode, FreightTimelineAction } from '../project/pipelines/types';

import './freight-timeline.less';
import { headerButtonStyle } from './utils';

export const FreightTimelineHeader = ({
  promotingStage,
  action,
  cancel,
  downstreamSubs,
  selectedWarehouse,
  setSelectedWarehouse,
  warehouses,
  collapsed,
  setCollapsed,
  collapsable
}: {
  promotingStage?: string;
  action?: FreightTimelineAction;
  cancel: () => void;
  downstreamSubs?: string[];
  selectedWarehouse: string;
  setSelectedWarehouse: (warehouse: string) => void;
  warehouses: { [key: string]: Warehouse };
  collapsed: CollapseMode;
  setCollapsed: (collapsed: CollapseMode) => void;
  collapsable?: boolean;
}) => {
  const { stageColorMap } = useContext(ColorContext);
  const { name: projectName } = useParams();

  const getIcon = (action: FreightTimelineAction) => {
    switch (action) {
      case FreightTimelineAction.Promote:
        return faBullseye;
      case FreightTimelineAction.PromoteSubscribers:
        return faTruckArrowRight;
      case FreightTimelineAction.ManualApproval:
        return faCircleCheck;
      default:
        return faQuestionCircle;
    }
  };

  const navigate = useNavigate();

  const WarehouseSelector = (
    <Select
      className='w-48 ml-4'
      value={selectedWarehouse}
      onChange={(value) => setSelectedWarehouse(value)}
      size='small'
      labelRender={({ label }) => <div className='text-xs font-medium'>{label}</div>}
      optionRender={(opt) => (
        <div className='text-xs font-normal w-full h-full flex items-center'>
          <div>{opt.label}</div>
        </div>
      )}
      options={[
        ...(Object.keys(warehouses) || []).map((w) => ({ value: w, label: w })),
        {
          value: '',
          label: 'All warehouses'
        }
      ]}
    />
  );

  return (
    <div className='w-full pl-6 flex items-center font-semibold text-sm h-8 pt-2'>
      {action ? (
        <>
          <div className='flex items-center uppercase'>
            <FontAwesomeIcon icon={getIcon(action)} className='mr-2' />
            {promotingStage && action != 'manualApproval' ? (
              <>
                Promoting{' '}
                {action === 'promoteSubscribers'
                  ? `TO ${(downstreamSubs || []).length} Downstream Subscribers (${downstreamSubs?.join(', ')}) of`
                  : ''}{' '}
                Stage :{' '}
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
                    className='cursor-pointer text-gray-500 ml-2'
                  />
                </Tooltip>
              </>
            ) : (
              <>
                {action === 'manualApproval' ? 'Manually Approving Freight' : 'Promoting Freight'}
              </>
            )}
          </div>

          {(Object.keys(warehouses) || []).length > 1 && (
            <div className='ml-4'>
              <FontAwesomeIcon icon={faBuilding} className='mr-2' /> WAREHOUSE
              {WarehouseSelector}
            </div>
          )}

          <div
            className='ml-auto mr-4 cursor-pointer px-2 py-1 text-white bg-gray-700 rounded hover:bg-gray-600 font-semibold text-xs'
            onClick={cancel}
          >
            CANCEL
          </div>
        </>
      ) : (
        <>
          <div className='flex items-center text-gray-400 text-xs mr-auto ml-4'>
            <FontAwesomeIcon icon={faTimeline} className='mr-2' />
            FREIGHT TIMELINE
          </div>
          {collapsable && (
            <>
              <Tooltip
                title={`${collapsed === CollapseMode.HideAll ? 'Expand' : 'Collapse'} unused freight`}
              >
                <Button
                  icon={
                    <FontAwesomeIcon
                      icon={collapsed === CollapseMode.HideAll ? faExpand : faCompress}
                    />
                  }
                  size='small'
                  className={headerButtonStyle(collapsed === CollapseMode.HideAll)}
                  onClick={() =>
                    setCollapsed(
                      collapsed === CollapseMode.HideAll
                        ? CollapseMode.Expanded
                        : CollapseMode.HideAll
                    )
                  }
                />
              </Tooltip>
              <Tooltip
                title={`${collapsed === CollapseMode.HideOld ? 'Show' : 'Hide'} old freight`}
              >
                <Button
                  icon={
                    <FontAwesomeIcon
                      icon={collapsed === CollapseMode.HideOld ? faCalendarPlus : faCalendarXmark}
                    />
                  }
                  size='small'
                  className={headerButtonStyle(collapsed === CollapseMode.HideOld)}
                  onClick={() =>
                    setCollapsed(
                      collapsed === CollapseMode.HideOld
                        ? CollapseMode.Expanded
                        : CollapseMode.HideOld
                    )
                  }
                />
              </Tooltip>
            </>
          )}
          <Tooltip title='Assemble Freight' placement='left'>
            <Button
              icon={<FontAwesomeIcon icon={faTools} />}
              size='small'
              className={headerButtonStyle(false)}
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
            />
          </Tooltip>
          {(Object.keys(warehouses) || []).length > 1 && (
            <div className='mr-4 -mb-1'>{WarehouseSelector}</div>
          )}
        </>
      )}
    </div>
  );
};

export default FreightTimelineHeader;
