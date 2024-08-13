import { useMutation } from '@connectrpc/connect-query';
import { faBook, faCode, faListCheck, faTheaterMasks } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Col, Drawer, Flex, Input, Row, Tabs, Typography } from 'antd';
import type { JSONSchema4 } from 'json-schema';
import { Controller, useForm } from 'react-hook-form';
import { generatePath, useNavigate } from 'react-router-dom';
import { z } from 'zod';

import { paths } from '@ui/config/paths';
import { YamlEditor } from '@ui/features/common/code-editor/yaml-editor';
import { FieldContainer } from '@ui/features/common/form/field-container';
import schema from '@ui/gen/schema/stages.kargo.akuity.io_v1alpha1.json';
import { createResource } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { zodValidators } from '@ui/utils/validators';

import { getStageYAMLExample } from '../project/pipelines/utils/stage-yaml-example';

import { GitUpdateEditor } from './git-update-editor/git-update-editor';
import { requestedFreightSchema } from './git-update-editor/schemas';
import { RequestedFreight } from './requested-freight';
import { RequestedFreightEditor } from './requested-freight-editor';

const formSchema = z.object({
  value: zodValidators.requiredString
});

const wizardSchema = z.object({
  name: zodValidators.requiredString,
  requestedFreight: z.array(requestedFreightSchema)
});

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

  const { mutateAsync, isPending } = useMutation(createResource, {
    onSuccess: () => close()
  });

  const { control, handleSubmit } = useForm({
    defaultValues: {
      value: getStageYAMLExample(project || '')
    },
    resolver: zodResolver(formSchema)
  });

  const {
    control: wizardControl,
    handleSubmit: wizardSubmit,
    watch
  } = useForm({
    defaultValues: {
      name: '',
      requestedFreight: []
    },
    resolver: zodResolver(wizardSchema)
  });

  const onSubmit = handleSubmit(async (data) => {
    const textEncoder = new TextEncoder();
    await mutateAsync({
      manifest: textEncoder.encode(data.value)
    });
  });

  if (!project) {
    return;
  }

  const requestedFreight = watch('requestedFreight');

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

      <Tabs>
        <Tabs.TabPane key='1' tab='Form' icon={<FontAwesomeIcon icon={faListCheck} />}>
          <FieldContainer name='name' label='Name' control={wizardControl}>
            {({ field }) => <Input {...field} placeholder='my-stage' />}
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

          <GitUpdateEditor />
        </Tabs.TabPane>
        <Tabs.TabPane key='2' tab='YAML' icon={<FontAwesomeIcon icon={faCode} />}>
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
