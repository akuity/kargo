import {
  faChevronDown,
  faCircleNodes,
  faIdBadge,
  faMasksTheater,
  faPalette,
  faWandSparkles,
  faWarehouse
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Dropdown, Space, Tabs, Tooltip } from 'antd';
import { useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import { useModal } from '@ui/features/common/modal/use-modal';
import { CredentialsList } from '@ui/features/project/credentials/credentials-list';
import { CreateStageModal } from '@ui/features/project/project-details/create-stage-modal';
import { CreateWarehouseModal } from '@ui/features/project/project-details/create-warehouse-modal';
import { ProjectDetails } from '@ui/features/project/project-details/project-details';
import { clearColors } from '@ui/features/stage/utils';

export type ProjectTabs = 'details' | 'credentials';

const tabToKey = (tab: ProjectTabs) => {
  switch (tab) {
    case 'details':
      return '1';
    case 'credentials':
      return '2';
  }
};

export const Project = ({ tab }: { tab?: ProjectTabs }) => {
  const { name } = useParams();
  const navigate = useNavigate();

  const { show: showCreateStage } = useModal(
    name ? (p) => <CreateStageModal {...p} project={name} /> : undefined
  );
  const { show: showCreateWarehouse } = useModal(
    name ? (p) => <CreateWarehouseModal {...p} project={name} /> : undefined
  );

  const [activeTab, setActiveTab] = useState(tabToKey(tab || 'details'));
  // we must render the tab contents outside of the Antd tabs component to prevent layout issues in the ProjectDetails component
  const renderTab = (key: string) => {
    switch (key) {
      case '1':
        return <ProjectDetails />;
      case '2':
        return <CredentialsList />;
      default:
        return <ProjectDetails />;
    }
  };

  return (
    <div className='h-full flex flex-col'>
      <div className='p-6'>
        <div className='flex items-center'>
          <div className='mr-auto'>
            <div className='font-semibold mb-1 text-xs text-gray-600'>PROJECT</div>
            <div className='text-2xl font-semibold'>{name}</div>
          </div>
          <Tooltip title='Reassign Stage Colors'>
            <Button
              type='default'
              className='mr-2'
              onClick={() => {
                clearColors(name || '');
                window.location.reload();
              }}
            >
              <FontAwesomeIcon icon={faPalette} />
            </Button>
          </Tooltip>{' '}
          <Dropdown
            menu={{
              items: [
                {
                  key: '1',
                  label: (
                    <>
                      <FontAwesomeIcon icon={faMasksTheater} size='xs' className='mr-2' /> Stage
                    </>
                  ),
                  onClick: () => showCreateStage()
                },
                {
                  key: '2',
                  label: (
                    <>
                      <FontAwesomeIcon icon={faWarehouse} size='xs' className='mr-2' /> Warehouse
                    </>
                  ),
                  onClick: () => showCreateWarehouse()
                }
              ]
            }}
            placement='bottomRight'
            trigger={['click']}
          >
            <Button type='primary' icon={<FontAwesomeIcon icon={faWandSparkles} size='1x' />}>
              <Space>
                Create
                <FontAwesomeIcon icon={faChevronDown} size='xs' />
              </Space>
            </Button>
          </Dropdown>
        </div>
      </div>
      <Tabs
        defaultActiveKey='1'
        activeKey={activeTab}
        onChange={(k) => {
          setActiveTab(k);
          navigate(`/project/${name}${k === '1' ? '' : '/credentials'}`);
        }}
        tabBarStyle={{
          padding: '0 24px',
          marginBottom: '0.5rem'
        }}
        items={[
          {
            key: '1',
            label: (
              <>
                <FontAwesomeIcon icon={faCircleNodes} className='mr-2' />
                Details
              </>
            )
          },
          {
            key: '2',
            label: (
              <>
                <FontAwesomeIcon icon={faIdBadge} className='mr-2' />
                Credentials
              </>
            )
          }
        ]}
      />
      {renderTab(activeTab)}
    </div>
  );
};
