import { useQuery } from '@connectrpc/connect-query';
import {
  faArrowDownShortWide,
  faBuilding,
  faFileLines,
  faGear,
  faTools
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Drawer, Flex, Skeleton, Tabs, Typography } from 'antd';
import Alert from 'antd/es/alert/Alert';
import { useMemo } from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { AssembleFreight } from '@ui/features/assemble-freight/assemble-freight';
import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import { getWarehouse } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { RawFormat } from '@ui/gen/api/service/v1alpha1/service_pb';
import { Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';
import { decodeRawData } from '@ui/utils/decode-raw-data';

import { RepoSubscriptions } from './repo-subscriptions';
import { WarehouseSettings } from './tabs/settings/warehouse-settings';
import { getWarehouseError } from './warehouse-error';

export const WarehouseDetails = ({
  warehouse,
  refetchFreight
}: {
  warehouse: Warehouse;
  refetchFreight: () => void;
}) => {
  const { name: projectName, warehouseName, tab } = useParams();
  const navigate = useNavigate();

  const onClose = () => navigate(generatePath(paths.project, { name: projectName }));

  const warehouseErrorMessage = useMemo(() => getWarehouseError(warehouse), [warehouse]);

  const rawWarehouseYamlQuery = useQuery(getWarehouse, {
    project: projectName,
    name: warehouseName,
    format: RawFormat.YAML
  });

  const rawWarehouseYaml = useMemo(
    () => decodeRawData(rawWarehouseYamlQuery.data),
    [rawWarehouseYamlQuery.data]
  );

  return (
    <Drawer
      open={!!warehouse}
      onClose={onClose}
      width={'80%'}
      title={
        <Flex justify='space-between' className='font-normal'>
          <div>
            <Flex gap={16} align='center'>
              <Typography.Title level={2} style={{ margin: 0 }}>
                <FontAwesomeIcon icon={faBuilding} size='xs' className='mr-1 text-gray-400' />{' '}
                {warehouse.metadata?.name}
              </Typography.Title>
            </Flex>
            <div className='-mt-1'>
              <Typography.Text type='secondary'>{projectName}</Typography.Text>
            </div>
          </div>
        </Flex>
      }
    >
      {warehouse && (
        <div className='flex flex-col h-full'>
          {warehouseErrorMessage && (
            <Alert className='mb-6' message={warehouseErrorMessage} type='error' closable />
          )}

          <Tabs
            className='-mt-4'
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
              children={
                rawWarehouseYamlQuery.isLoading ? (
                  <Skeleton />
                ) : (
                  <YamlEditor
                    value={rawWarehouseYaml}
                    height='700px'
                    isHideManagedFieldsDisplayed
                    disabled
                  />
                )
              }
            />
            <Tabs.TabPane
              key='settings'
              tab='Settings'
              icon={<FontAwesomeIcon icon={faGear} />}
              children={<WarehouseSettings />}
            />
          </Tabs>
        </div>
      )}
    </Drawer>
  );
};

export default WarehouseDetails;
