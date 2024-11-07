import { useMutation } from '@connectrpc/connect-query';
import { faBook, faBoxes, faCode, faListCheck } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Drawer, Flex, Tabs, Typography } from 'antd';
import { JSONSchema4 } from 'json-schema';
import { useCallback, useState } from 'react';
import { useParams } from 'react-router-dom';
import { z } from 'zod';

import { YamlEditor } from '@ui/features/common/code-editor/yaml-editor';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { WarehouseManifestsGen } from '@ui/features/utils/manifest-generator';
import { useSearchParamsState } from '@ui/features/utils/use-search-params-state';
import warehouseSchema from '@ui/gen/schema/warehouses.kargo.akuity.io_v1alpha1.json';
import { createResource } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

import { CreateWarehouseWizard } from './create-warehouse-wizard';

const urlStateSchema = z.object({
  tab: z.enum(['wizard', 'yaml']).catch('wizard'),
  state: z.record(z.string(), z.any()).catch({})
});

const Body = () => {
  const { name: projectName } = useParams();

  const urlState = useSearchParamsState(urlStateSchema);

  if (!projectName) {
    throw new Error(`Expected project name in URL`);
  }

  const createResourceMutation = useMutation(createResource, {
    onSuccess: () => {
      urlState.removeKeysFromSearch(['create', 'tab', 'state']);
    }
  });

  const tab = urlState.state.tab;

  const getWarehouseManifest = useCallback(() => {
    const manifest = urlState.state.state;
    if (manifest) {
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
  }, [urlState.state.state]);

  const [yaml, setYaml] = useState(getWarehouseManifest);

  const onSubmit = useCallback(
    () =>
      createResourceMutation.mutate({
        manifest: new TextEncoder().encode(getWarehouseManifest())
      }),
    [getWarehouseManifest, createResourceMutation.mutate]
  );

  const formState = urlState.state.state;

  const setFormState = (nextState: object) => urlState.setSearchState({ state: nextState });

  return (
    <>
      <Tabs
        activeKey={tab}
        onChange={(newTab) => {
          if (tab === 'wizard' && newTab === 'yaml') {
            setYaml(getWarehouseManifest());
          }
          urlState.setSearchState({ tab: newTab as 'wizard' | 'yaml' });
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
