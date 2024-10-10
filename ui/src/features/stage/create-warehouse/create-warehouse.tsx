import { useMutation } from '@connectrpc/connect-query';
import { faBook, faBoxes, faCode, faListCheck } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import Form from '@rjsf/antd';
import validator from '@rjsf/validator-ajv8';
import { Button, Checkbox, Drawer, Flex, Input, Tabs, Typography } from 'antd';
import AntdFormLabel from 'antd/es/form/FormItemLabel';
import { JSONSchema4, JSONSchema7 } from 'json-schema';
import { useCallback, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';

import { YamlEditor } from '@ui/features/common/code-editor/yaml-editor';
import { DescriptionFieldTemplate } from '@ui/features/common/form/rjsf/description-field-template';
import { FieldTemplate } from '@ui/features/common/form/rjsf/field-template';
import { ObjectFieldTemplate } from '@ui/features/common/form/rjsf/object-field-template';
import rjsfStylesOverride from '@ui/features/common/form/rjsf/style-overrides.module.less';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { WarehouseManifestsGen } from '@ui/features/utils/manifest-generator';
import { URLStates } from '@ui/features/utils/url-query-state/states';
import { useURLQueryState } from '@ui/features/utils/url-query-state/use-url-query-state';
import warehouseSchema from '@ui/gen/schema/warehouses.kargo.akuity.io_v1alpha1.json';
import { createResource } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

import { warehouseCreateFormJSONSchema } from './schema';

const Wizard = () => {
  const [showDescription, setShowDescription] = useState(false);

  const [state, setState] = useURLQueryState<URLStates['project']>();

  const formState = useMemo(() => {
    return JSON.parse(state?.state || '{}');
  }, [state?.state]);

  const setFormState = (nextState: object) =>
    setState({ ...state, state: JSON.stringify(nextState) });

  return (
    <>
      <div className='flex'>
        <Checkbox
          className='text-xs mb-2 ml-auto'
          checked={showDescription}
          onChange={(e) => setShowDescription(e.target.checked)}
        >
          Show description
        </Checkbox>
      </div>
      <div className={rjsfStylesOverride.container}>
        <AntdFormLabel prefixCls='' label='Name' htmlFor='warehouse-name' />
        <Input
          aria-label='warehouse-name'
          value={formState?.name || ''}
          onChange={(e) => setFormState({ ...formState, name: e.target.value })}
          className='mb-4'
        />
        <Form
          key={`${showDescription}`}
          formData={formState?.spec || {}}
          onChange={({ formData }) => setFormState({ ...formState, spec: formData })}
          schema={warehouseCreateFormJSONSchema as JSONSchema7}
          validator={validator}
          templates={{
            ObjectFieldTemplate,
            DescriptionFieldTemplate: showDescription ? DescriptionFieldTemplate : () => null,
            ArrayFieldDescriptionTemplate: () => null,
            FieldTemplate
          }}
          uiSchema={{
            'ui:submitButtonOptions': {
              norender: true
            }
          }}
        />
      </div>
    </>
  );
};

const Body = () => {
  const { name: projectName } = useParams();

  if (!projectName) {
    throw new Error(`Expected project name in URL`);
  }

  const [urlQuery, setURLQuery] = useURLQueryState<URLStates['project']>();

  const createResourceMutation = useMutation(createResource, {
    onSuccess: () => {
      setURLQuery();
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
            children: <Wizard />
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
