import { Code, ConnectError } from '@connectrpc/connect';
import { useMutation, useQuery } from '@connectrpc/connect-query';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Card, Flex, message, notification } from 'antd';
import type { JSONSchema4 } from 'json-schema';
import { useForm } from 'react-hook-form';
import { useParams } from 'react-router-dom';
import yaml from 'yaml';
import { z } from 'zod';

import { newErrorHandler, newTransportWithAuth } from '@ui/config/transport';
import { YamlEditor } from '@ui/features/common/code-editor/yaml-editor';
import { FieldContainer } from '@ui/features/common/form/field-container';
import {
  projectConfigYAMLExample,
  projectYAMLExample
} from '@ui/features/project/list/utils/project-yaml-example';
import {
  createOrUpdateResource,
  getProject,
  getProjectConfig,
  updateResource
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { RawFormat } from '@ui/gen/api/service/v1alpha1/service_pb';
import projectConfigSchema from '@ui/gen/schema/projectconfigs.kargo.akuity.io_v1alpha1.json';
import schema from '@ui/gen/schema/projects.kargo.akuity.io_v1alpha1.json';
import { decodeRawData } from '@ui/utils/decode-raw-data';
import { zodValidators } from '@ui/utils/validators';

const formSchema = z.object({
  value: zodValidators.requiredString
});

import { DeleteProject } from './delete-project-modal';

const transport = newTransportWithAuth(
  newErrorHandler((err) => {
    if (err.code === Code.NotFound) {
      // ignore
      return;
    }

    const errorMessage = err instanceof ConnectError ? err.rawMessage : 'Unexpected API error';
    notification.error({ message: errorMessage, placement: 'bottomRight' });
  })
);

export const GeneralSettings = () => {
  const { name } = useParams();
  const { data, isLoading } = useQuery(getProject, { name, format: RawFormat.YAML });

  const projectConfigQuery = useQuery(
    getProjectConfig,
    { name, format: RawFormat.YAML },
    {
      transport
    }
  );

  const projectConfigYAML = decodeRawData(projectConfigQuery.data);

  const projectConfigForm = useForm({
    values: {
      value: projectConfigYAML
    },
    resolver: zodResolver(formSchema)
  });

  const { mutateAsync, isPending } = useMutation(updateResource, {
    onSuccess: () =>
      message.success({
        content: `Project Configuration has been updated.`
      })
  });

  const createOrUpdateMutation = useMutation(createOrUpdateResource, {
    onSuccess: () => message.success({ content: `ProjectConfig has been updated.` })
  });

  const { control, handleSubmit } = useForm({
    values: {
      value: decodeRawData(data)
    },
    resolver: zodResolver(formSchema)
  });

  const onSubmit = handleSubmit(async (data) => {
    const textEncoder = new TextEncoder();
    await mutateAsync({
      manifest: textEncoder.encode(data.value)
    });
  });

  const onSubmitConfig = projectConfigForm.handleSubmit(async (data) => {
    const textEncoder = new TextEncoder();
    await createOrUpdateMutation.mutateAsync({
      manifest: textEncoder.encode(data.value)
    });
  });

  return (
    <Flex gap={16} vertical>
      <Card title='General' type='inner'>
        <FieldContainer name='value' control={control}>
          {({ field: { value, onChange } }) => (
            <YamlEditor
              label='YAML'
              value={value}
              onChange={(e) => onChange(e || '')}
              height='500px'
              schema={schema as JSONSchema4}
              placeholder={yaml.stringify(projectYAMLExample)}
              isLoading={isLoading}
              resourceType='projects'
              isHideManagedFieldsDisplayed
            />
          )}
        </FieldContainer>
        <Flex justify='flex-end'>
          <Button type='primary' onClick={onSubmit} loading={isPending}>
            Update
          </Button>
        </Flex>
      </Card>

      <Card title='ProjectConfig' type='inner'>
        <FieldContainer control={projectConfigForm.control} name='value'>
          {({ field }) => (
            <YamlEditor
              label='YAML'
              isLoading={projectConfigQuery.isFetching}
              height='500px'
              value={field.value}
              onChange={(e) => field.onChange(e || '')}
              placeholder={yaml.stringify(projectConfigYAMLExample)}
              schema={projectConfigSchema as JSONSchema4}
              isHideManagedFieldsDisplayed
            />
          )}
        </FieldContainer>
        <Flex justify='flex-end'>
          <Button
            type='primary'
            onClick={onSubmitConfig}
            loading={createOrUpdateMutation.isPending}
          >
            {!projectConfigYAML ? 'Create' : 'Update'}
          </Button>
        </Flex>
      </Card>

      <Card title='Delete Project' type='inner'>
        <DeleteProject />
      </Card>
    </Flex>
  );
};
