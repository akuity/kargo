import { useMutation } from '@connectrpc/connect-query';
import {
  faBook,
  faCode,
  faListCheck,
  faTheaterMasks,
  faTimes
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Col, Drawer, Flex, Input, Row, Select, Tabs, Typography } from 'antd';
import type { JSONSchema4 } from 'json-schema';
import { useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { generatePath, useNavigate } from 'react-router-dom';
import yaml from 'yaml';
import { z } from 'zod';

import { paths } from '@ui/config/paths';
import { YamlEditor } from '@ui/features/common/code-editor/yaml-editor';
import { FieldContainer } from '@ui/features/common/form/field-container';
import schema from '@ui/gen/schema/stages.kargo.akuity.io_v1alpha1.json';
import { createResource } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { queryCache } from '@ui/utils/cache';
import { decodeUint8ArrayYamlManifestToJson } from '@ui/utils/decode-raw-data';
import { zodValidators } from '@ui/utils/validators';

import { promoStepsExample } from '../project/pipelines/utils/promotion-steps-example';
import { getStageYAMLExample } from '../project/pipelines/utils/stage-yaml-example';

import { requestedFreightSchema } from './git-update-editor/schemas';
import { RequestedFreight } from './requested-freight';
import { RequestedFreightEditor } from './requested-freight-editor';
import { ColorMapHex } from './utils';

const formSchema = z.object({
  value: zodValidators.requiredString
});

const wizardSchema = z.object({
  name: zodValidators.requiredString,
  requestedFreight: z.array(requestedFreightSchema),
  promotionMechanisms: z.string().optional(),
  color: z.string().optional(),
  // next step is to wizardify this
  promotionTemplateSteps: z.string().optional()
});

const stageFormToYAML = (data: z.infer<typeof wizardSchema>, namespace: string) => {
  return yaml.stringify({
    kind: 'Stage',
    apiVersion: 'kargo.akuity.io/v1alpha1',
    metadata: {
      name: data.name,
      namespace,
      ...(data.color &&
        data.color !== '' && {
          annotations: {
            'kargo.akuity.io/color': data.color
          }
        })
    },
    spec: {
      requestedFreight: data.requestedFreight,
      ...(data.promotionMechanisms && {
        promotionMechanisms: yaml.parse(data.promotionMechanisms)
      }),
      ...(data.promotionTemplateSteps && {
        promotionTemplate: { spec: { steps: yaml.parse(data.promotionTemplateSteps) } }
      })
    }
  });
};

export const CreateStage = ({
  project,
  warehouses,
  stages
}: {
  project?: string;
  warehouses?: string[];
  stages?: string[];
}) => {
  const navigate = useNavigate();
  const close = () => navigate(generatePath(paths.project, { name: project }));
  const [tab, setTab] = useState('wizard');

  const { mutateAsync, isPending } = useMutation(createResource, {
    onSuccess: (response) => {
      for (const result of response?.results || []) {
        if (result?.result?.case === 'createdResourceManifest') {
          queryCache.stage.add(project || '', [
            decodeUint8ArrayYamlManifestToJson(result?.result?.value)
          ]);
        }
      }
      close();
    }
  });

  const { control, handleSubmit, setValue } = useForm({
    defaultValues: {
      value: getStageYAMLExample(project || '')
    },
    resolver: zodResolver(formSchema)
  });

  const {
    control: wizardControl,
    watch,
    setValue: setWizardValue
  } = useForm({
    defaultValues: {
      name: '',
      requestedFreight: [],
      promotionMechanisms: '',
      color: undefined,
      promotionTemplateSteps: ''
    },
    resolver: zodResolver(wizardSchema)
  });

  const onSubmit = handleSubmit(async (data) => {
    let value = data.value;
    if (tab === 'wizard') {
      const unmarshalled = stageFormToYAML(watch(), project || '');
      setValue('value', unmarshalled);
      value = unmarshalled;
    }
    const textEncoder = new TextEncoder();
    await mutateAsync({
      manifest: textEncoder.encode(value)
    });
  });

  if (!project) {
    return;
  }

  const requestedFreight = watch('requestedFreight');
  const color = watch('color');

  return (
    <Drawer open={!!project} width={'80%'} closable={false} onClose={close}>
      <Flex align='center' className='mb-4'>
        <Typography.Title level={1} className='flex items-center !m-0'>
          <FontAwesomeIcon icon={faTheaterMasks} className='mr-2 text-base text-gray-400' />
          Create Stage
        </Typography.Title>
        <Typography.Link
          href='https://kargo.akuity.io/concepts/#stage-resources'
          target='_blank'
          className='ml-3'
        >
          <FontAwesomeIcon icon={faBook} />
        </Typography.Link>
        <Button onClick={close} className='ml-auto'>
          Cancel
        </Button>
      </Flex>

      <Tabs
        onChange={(newTab) => {
          if (tab === 'wizard' && newTab === 'yaml') {
            setValue('value', stageFormToYAML(watch(), project || ''));
          }
          setTab(newTab);
        }}
      >
        <Tabs.TabPane key='wizard' tab='Form' icon={<FontAwesomeIcon icon={faListCheck} />}>
          <FieldContainer name='name' label='Name' control={wizardControl}>
            {({ field }) => <Input {...field} placeholder='my-stage' />}
          </FieldContainer>
          <FieldContainer name='color' label='Color' control={wizardControl}>
            {({ field }) => (
              <Flex className='w-full' wrap>
                <Select
                  {...field}
                  placeholder='Select a color (optional)'
                  className='w-full shrink-0'
                  options={Object.keys(ColorMapHex).map((value) => {
                    return {
                      value,
                      label: (
                        <Flex align='center'>
                          <div
                            className='mr-2 rounded'
                            style={{
                              backgroundColor: ColorMapHex[value],
                              width: '10px',
                              height: '10px'
                            }}
                          />
                          {value.charAt(0).toUpperCase() + value.slice(1)}
                        </Flex>
                      )
                    };
                  })}
                />
                {color && color !== '' && (
                  <Button
                    onClick={() => setWizardValue('color', undefined)}
                    size='small'
                    danger
                    className='mt-2 ml-auto'
                    icon={<FontAwesomeIcon icon={faTimes} />}
                  >
                    Clear Color
                  </Button>
                )}
              </Flex>
            )}
          </FieldContainer>

          <Typography.Title level={4}>Requested Freight</Typography.Title>

          <Controller
            name='requestedFreight'
            control={wizardControl}
            render={({ field }) => (
              <Row className='mb-6' gutter={12}>
                <Col span={12}>
                  {requestedFreight?.length > 0 ? (
                    <RequestedFreight
                      requestedFreight={requestedFreight}
                      projectName={project}
                      className='mb-4 grid grid-cols-2 gap-4'
                      onDelete={(index) => {
                        field.onChange([
                          ...field.value.slice(0, index),
                          ...field.value.slice(index + 1)
                        ]);
                      }}
                    />
                  ) : (
                    <Flex
                      className='w-full h-full rounded-md bg-gray-50 text-gray-400 font-medium text-center'
                      align='center'
                      justify='center'
                    >
                      Requested Freight are required to create a Stage.
                      <br />
                      Add a Freight Request using the form to the right to continue.
                    </Flex>
                  )}
                </Col>
                <Col span={12}>
                  <RequestedFreightEditor
                    warehouses={warehouses}
                    stages={stages}
                    onSubmit={(value) => {
                      field.onChange([...field.value, value]);
                    }}
                  />
                </Col>
              </Row>
            )}
          />

          <Typography.Title level={4}>Promotion Steps</Typography.Title>
          <FieldContainer name='promotionTemplateSteps' control={wizardControl}>
            {({ field: { value, onChange } }) => (
              <YamlEditor
                value={value as string}
                onChange={(e) => onChange(e || '')}
                height='250px'
                schema={
                  (schema as JSONSchema4).properties?.spec.properties?.promotionTemplate?.properties
                    ?.spec?.properties?.steps
                }
                resourceType='stages'
                placeholder={promoStepsExample}
              />
            )}
          </FieldContainer>
        </Tabs.TabPane>
        <Tabs.TabPane key='yaml' tab='YAML' icon={<FontAwesomeIcon icon={faCode} />}>
          <FieldContainer name='value' control={control}>
            {({ field: { value, onChange } }) => (
              <YamlEditor
                value={value}
                onChange={(e) => onChange(e || '')}
                height='500px'
                schema={schema as JSONSchema4}
                placeholder={getStageYAMLExample(project)}
                resourceType='stages'
              />
            )}
          </FieldContainer>
        </Tabs.TabPane>
      </Tabs>
      <Button onClick={onSubmit} loading={isPending}>
        Create
      </Button>
    </Drawer>
  );
};

export default CreateStage;
