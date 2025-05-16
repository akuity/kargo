import { useMutation } from '@connectrpc/connect-query';
import { faCode, faListCheck } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Drawer, Tabs, Typography } from 'antd';
import { JSONSchema4 } from 'json-schema';
import { useCallback, useState } from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { YamlEditor } from '@ui/features/common/code-editor/yaml-editor';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { warehouseManifestsGen } from '@ui/features/utils/manifest-generator';
import { createResource } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import warehouseSchema from '@ui/gen/schema/warehouses.kargo.akuity.io_v1alpha1.json';

import { CreateWarehouseWizard } from './create-warehouse-wizard';

const Body = () => {
  const { name: projectName } = useParams();
  const navigate = useNavigate();

  if (!projectName) {
    throw new Error(`Expected project name in URL`);
  }

  const createResourceMutation = useMutation(createResource, {
    onSuccess: () => {
      navigate(generatePath(paths.project, { name: projectName }));
    }
  });

  const [tab, setTab] = useState<'wizard' | 'yaml'>('wizard');

  const [form, setForm] = useState<object>();

  const getWarehouseManifest = useCallback(() => {
    const manifest = form;
    if (manifest) {
      return warehouseManifestsGen.v1alpha1({
        projectName,
        // @ts-expect-error correct values from dynamic form
        warehouseName: manifest.name,
        // @ts-expect-error correct values from dynamic form
        spec: manifest.spec
      });
    }

    return warehouseManifestsGen.v1alpha1({
      projectName,
      warehouseName: '',
      spec: {
        subscriptions: []
      }
    });
  }, [form]);

  const [yaml, setYaml] = useState(getWarehouseManifest);

  const onSubmit = useCallback(
    () =>
      createResourceMutation.mutate({
        manifest: new TextEncoder().encode(getWarehouseManifest())
      }),
    [getWarehouseManifest, createResourceMutation.mutate]
  );

  return (
    <>
      <Tabs
        activeKey={tab}
        className='-mt-4'
        onChange={(newTab) => {
          if (tab === 'wizard' && newTab === 'yaml') {
            setYaml(getWarehouseManifest());
          }
          setTab(newTab as 'wizard' | 'yaml');
        }}
        items={[
          {
            key: 'wizard',
            icon: <FontAwesomeIcon icon={faListCheck} />,
            label: 'Form',
            children: (
              <CreateWarehouseWizard
                formState={(form || {}) as Record<string, unknown>}
                setFormState={(next) => setForm(next)}
              />
            )
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

      <Button
        className='mt-5'
        loading={createResourceMutation.isPending}
        onClick={onSubmit}
        type='primary'
      >
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
      width='80%'
      title='Create Warehouse'
      extra={
        <Typography.Link
          href='https://docs.kargo.io/user-guide/how-to-guides/working-with-warehouses'
          target='_blank'
          className='ml-3'
        >
          Docs
        </Typography.Link>
      }
    >
      {props.visible && <Body />}
    </Drawer>
  );
};

export default CreateWarehouse;
