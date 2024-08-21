import {
  faArrowDownShortWide,
  faBuilding,
  faFileLines,
  faTools
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Drawer, Tabs, Typography } from 'antd';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { AssembleFreight } from '@ui/features/assemble-freight/assemble-freight';
import { Warehouse } from '@ui/gen/v1alpha1/generated_pb';

import { EditWarehouse } from './edit-warehouse';
import { RepoSubscriptions } from './repo-subscriptions';
import { WarehouseActions } from './warehouse-actions';

export const WarehouseDetails = ({
  warehouse,
  refetchFreight
}: {
  warehouse: Warehouse;
  refetchFreight: () => void;
}) => {
  const { name: projectName, tab } = useParams();
  const navigate = useNavigate();

  const onClose = () => navigate(generatePath(paths.project, { name: projectName }));

  return (
    <Drawer open={!!warehouse} onClose={onClose} width={'80%'} closable={false}>
      {warehouse && (
        <div className='flex flex-col h-full'>
          <div className='flex items-center justify-between mb-4'>
            <div className='flex gap-1 items-start'>
              <div>
                <div className='font-medium text-xs text-gray-500'>WAREHOUSE</div>
                <Typography.Title level={1} className='flex items-center m-0 !mb-2'>
                  <FontAwesomeIcon icon={faBuilding} className='mr-2 text-base text-gray-400' />
                  {warehouse.metadata?.name}
                </Typography.Title>
                <Typography.Text type='secondary'>
                  <span className='uppercase text-xs'>Project: </span>
                  <span className='font-semibold'>{projectName}</span>
                </Typography.Text>
              </div>
            </div>
            <WarehouseActions warehouse={warehouse} />
          </div>

          <Tabs
            defaultActiveKey='1'
            activeKey={tab}
            onChange={(tab) => {
              navigate(
                generatePath(paths.warehouse, {
                  name: projectName,
                  warehouseName: warehouse?.metadata?.name,
                  tab
                })
              );
            }}
          >
            <Tabs.TabPane
              key='subscriptions'
              tab='Subscriptions'
              icon={<FontAwesomeIcon icon={faArrowDownShortWide} />}
            >
              <div className='flex flex-col gap-8 flex-1'>
                <RepoSubscriptions subscriptions={warehouse.spec?.subscriptions} />
              </div>
            </Tabs.TabPane>
            <Tabs.TabPane
              key='create-freight'
              tab='Freight Assembly'
              icon={<FontAwesomeIcon icon={faTools} />}
            >
              <AssembleFreight
                warehouse={warehouse}
                onSuccess={() => {
                  onClose();
                  refetchFreight();
                }}
              />
            </Tabs.TabPane>
            <Tabs.TabPane
              key='live-manifest'
              tab='Live Manifest'
              icon={<FontAwesomeIcon icon={faFileLines} />}
            >
              <EditWarehouse projectName={projectName} warehouseName={warehouse.metadata?.name} />
            </Tabs.TabPane>
          </Tabs>
        </div>
      )}
    </Drawer>
  );
};

export default WarehouseDetails;
