import { faFile, faInfoCircle } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Drawer, Tabs, Typography } from 'antd';
import classNames from 'classnames';
import { useEffect, useState } from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { Freight } from '@ui/gen/v1alpha1/generated_pb';

import { Description } from '../common/description';
import { ManifestPreview } from '../common/manifest-preview';
import { getAlias } from '../common/utils';

import { FreightStatusList } from './freight-status-list';

const CopyValue = (props: { value: string; label: string; className?: string }) => (
  <div className={classNames('flex items-center text-gray-500 font-mono', props.className)}>
    <span className='text-gray-400 mr-2 text-xs'>{props.label}</span>
    <Typography.Text copyable>{props.value}</Typography.Text>
  </div>
);
export const FreightDetails = ({ freight }: { freight?: Freight }) => {
  const navigate = useNavigate();
  const { name: projectName } = useParams();
  const [alias, setAlias] = useState<string | undefined>();

  useEffect(() => {
    if (freight) {
      setAlias(getAlias(freight));
    }
  }, [freight]);

  const onClose = () => navigate(generatePath(paths.project, { name: projectName }));

  return (
    <Drawer open={!!freight} onClose={onClose} width={'80%'} closable={false}>
      {freight && (
        <div className='flex flex-col h-full'>
          <div className='flex items-center justify-between mb-4'>
            <div>
              <Typography.Title level={1} style={{ margin: 0, marginBottom: '0.5em' }}>
                {alias || freight.metadata?.name}
              </Typography.Title>
              {alias && freight?.metadata?.name && (
                <CopyValue label='NAME:' value={freight.metadata?.name} />
              )}
              <Description item={freight} loading={false} className='mt-2' />
            </div>

            {freight?.metadata?.uid && <CopyValue label='UID:' value={freight?.metadata?.uid} />}
          </div>
          <div className='flex flex-col flex-1'>
            <Tabs
              className='flex-1'
              defaultActiveKey='1'
              style={{ minHeight: '500px' }}
              items={[
                {
                  key: '1',
                  label: 'Details',
                  icon: <FontAwesomeIcon icon={faInfoCircle} />,
                  children: <FreightStatusList freight={freight} />
                },
                {
                  key: '2',
                  label: 'Live Manifest',
                  icon: <FontAwesomeIcon icon={faFile} />,
                  className: 'h-full pb-2',
                  children: <ManifestPreview object={freight} />
                }
              ]}
            />
          </div>
        </div>
      )}
    </Drawer>
  );
};
