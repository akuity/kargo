import { useMutation } from '@connectrpc/connect-query';
import { faBook, faBoxes, faCode, faListCheck } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Drawer, Flex, Tabs, Typography } from 'antd';
import { JSONSchema4 } from 'json-schema';
import { useCallback, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';

import { YamlEditor } from '@ui/features/common/code-editor/yaml-editor';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { WarehouseManifestsGen } from '@ui/features/utils/manifest-generator';
import { URLStates } from '@ui/features/utils/url-query-state/states';
import { useURLQueryState } from '@ui/features/utils/url-query-state/use-url-query-state';
import warehouseSchema from '@ui/gen/schema/warehouses.kargo.akuity.io_v1alpha1.json';
import { createResource } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

import { CreateWarehouseWizard } from './create-warehouse-wizard';

const Body = () => {
  const { name: projectName } = useParams();

  if (!projectName) {
    throw new Error(`Expected project name in URL`);
  }

  const [urlQuery, setURLQuery, clearState] = useURLQueryState<URLStates['project']>();

  const createResourceMutation = useMutation(createResource, {
    onSuccess: () => {
      clearState();
    }
  });

  const tab = urlQuery?.tab || 'wizard';

  const getWarehouseManifest = useCallback(() => {
    if (urlQuery?.state) {
      const manifest = JSON.parse(urlQuery?.state);
      return WarehouseManifestsGen.v1alpha1({
        projectName,
        warehouseName: manifest.name,
        spec: manifest.spec
      });
    }

    return WarehouseManifestsGen.v1alpha1({
      projectName,
      warehouseName: '',
      spec: {
        subscriptions: []
      }
    });
  }, [urlQuery?.state]);

  const [yaml, setYaml] = useState(getWarehouseManifest);

  const onSubmit = useCallback(
    () =>
      createResourceMutation.mutate({
        manifest: new TextEncoder().encode(getWarehouseManifest())
      }),
    [getWarehouseManifest, createResourceMutation.mutate]
  );

  const formState = useMemo(() => {
    return JSON.parse(urlQuery?.state || '{}');
  }, [urlQuery?.state]);

  const setFormState = (nextState: object) =>
    setURLQuery({ state: JSON.stringify(nextState) }, { replace: true });

  return (
    <>
      <Tabs
        activeKey={tab}
        onChange={(newTab) => {
          if (tab === 'wizard' && newTab === 'yaml') {
            setYaml(getWarehouseManifest());
          }
          setURLQuery({ ...urlQuery, tab: newTab as URLStates['project']['tab'] });
        }}
        items={[
          {
            key: 'wizard',
            icon: <FontAwesomeIcon icon={faListCheck} />,
            label: 'Form',
            children: <CreateWarehouseWizard formState={formState} setFormState={setFormState} />
          },
          {
            key: 'yaml',
            icon: <FontAwesomeIcon icon={faCode} />,
            label: 'YAML',
            children: (
              <YamlEditor
                schema={warehouseSchema as JSONSchema4}
                value={yaml}
                height='570px'
                onChange={(nextYaml) => setYaml(nextYaml || '')}
              />
            )
          }
        ]}
      />

      <Button className='mt-5' loading={createResourceMutation.isPending} onClick={onSubmit}>
        Create
      </Button>
    </>
  );
};

const CreateWarehouse = (props: ModalComponentProps) => {
  return (
    <Drawer
      open={props.visible}
      onClose={props.hide}
      width='60%'
      title={
        <Flex align='center'>
          <Typography.Title level={1} className='flex items-center !m-0'>
            <FontAwesomeIcon icon={faBoxes} className='mr-2 text-base text-gray-400' />
            Create Warehouse
          </Typography.Title>
          <Typography.Link
            href='https://docs.kargo.io/concepts/#warehouse-resources'
            target='_blank'
            className='ml-3'
          >
            <FontAwesomeIcon icon={faBook} />
          </Typography.Link>
        </Flex>
      }
    >
      {props.visible && <Body />}
    </Drawer>
  );
};

export default CreateWarehouse;
