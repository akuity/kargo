import { Divider, Drawer, Typography } from 'antd';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { Warehouse } from '@ui/gen/v1alpha1/generated_pb';

import { RepoSubscriptions } from './repo-subscriptions';
import { WarehouseActions } from './warehouse-actions';

export const WarehouseDetails = ({ warehouse }: { warehouse: Warehouse }) => {
  const { name: projectName } = useParams();
  const navigate = useNavigate();

  const onClose = () => navigate(generatePath(paths.project, { name: projectName }));

  return (
    <Drawer open={!!warehouse} onClose={onClose} width={'80%'} closable={false}>
      {warehouse && (
        <div className='flex flex-col h-full'>
          <div className='flex items-center justify-between'>
            <div className='flex gap-1 items-start'>
              <div>
                <Typography.Title level={1} style={{ margin: 0 }}>
                  {warehouse.metadata?.name}
                </Typography.Title>
                <Typography.Text type='secondary'>{projectName}</Typography.Text>
              </div>
            </div>
            <WarehouseActions warehouse={warehouse} />
          </div>
          <Divider style={{ marginTop: '1em' }} />

          <div className='flex flex-col gap-8 flex-1'>
            <RepoSubscriptions subscriptions={warehouse.spec?.subscriptions} />
          </div>
        </div>
      )}
    </Drawer>
  );
};
