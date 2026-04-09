import { faPlus } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Card, Flex, message, notification } from 'antd';
import type { JSONSchema4 } from 'json-schema';
import { useMemo } from 'react';
import { useForm } from 'react-hook-form';
import { useParams } from 'react-router-dom';
import yaml, { parse, stringify } from 'yaml';
import { z } from 'zod';

import { YamlEditor } from '@ui/features/common/code-editor/yaml-editor';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { useModal } from '@ui/features/common/modal/use-modal';
import { projectConfigYAMLExample } from '@ui/features/project/list/utils/project-yaml-example';
import { useGetProjectConfig } from '@ui/gen/api/v2/core/core';
import { useUpdateResource } from '@ui/gen/api/v2/resources/resources';
import projectConfigSchema from '@ui/gen/schema/projectconfigs.kargo.akuity.io_v1alpha1.json';
import { zodValidators } from '@ui/utils/validators';

import { Refresh } from './refresh';
import { CreateWebhookModal } from './webhook/create-webhook-modal';
import { Webhooks } from './webhooks';

const formSchema = z.object({
  value: zodValidators.requiredString
});

export const ProjectConfig = () => {
  const { name } = useParams();

  const projectConfigQuery = useGetProjectConfig(name || '', {
    query: { meta: { silent404: true } }
  });

  const projectConfigYAML = useMemo(() => {
    if (!projectConfigQuery.data?.data) {
      return '';
    }
    try {
      return stringify(projectConfigQuery.data.data);
    } catch (e) {
      notification.error({
        message: (e as Error)?.message || 'Failed to stringify ProjectConfig',
        placement: 'bottomRight'
      });
      return '';
    }
  }, [projectConfigQuery.data?.data]);

  const projectConfig = useMemo(() => {
    if (!projectConfigYAML) {
      return undefined;
    }
    try {
      return parse(projectConfigYAML);
    } catch (e) {
      notification.error({
        message: (e as Error)?.message || 'Failed to parse ProjectConfig YAML',
        placement: 'bottomRight'
      });
    }
  }, [projectConfigYAML]);

  const webhookReceivers = projectConfig?.status?.webhookReceivers || [];

  const creation = !projectConfigYAML;

  const projectConfigForm = useForm({
    values: {
      value: projectConfigYAML
    },
    resolver: zodResolver(formSchema)
  });

  const createOrUpdateMutation = useUpdateResource({
    mutation: {
      onSuccess: () => {
        message.success({ content: `ProjectConfig has been ${creation ? 'created' : 'updated'}` });
        projectConfigQuery.refetch();
      }
    }
  });

  const onSubmitConfig = projectConfigForm.handleSubmit((data) =>
    createOrUpdateMutation.mutate({ data: data.value })
  );

  const createWebhookModal = useModal((props) => (
    <CreateWebhookModal projectConfigYAML={projectConfigYAML} project={name || ''} {...props} />
  ));

  return (
    <Flex gap={16} vertical>
      <Card
        title='ProjectConfig'
        type='inner'
        extra={projectConfigYAML !== '' && <Refresh project={name || ''} />}
      >
        <FieldContainer control={projectConfigForm.control} name='value'>
          {({ field }) => (
            <YamlEditor
              label='YAML'
              isLoading={projectConfigQuery.isLoading}
              height='500px'
              value={field.value}
              onChange={(e) => field.onChange(e || '')}
              placeholder={yaml.stringify(projectConfigYAMLExample)}
              schema={projectConfigSchema as JSONSchema4}
            />
          )}
        </FieldContainer>
        <Flex>
          <Button
            icon={<FontAwesomeIcon icon={faPlus} />}
            onClick={() => createWebhookModal.show()}
          >
            Add Webhook
          </Button>
          <Button
            className='ml-auto'
            type='primary'
            onClick={onSubmitConfig}
            loading={createOrUpdateMutation.isPending}
          >
            {creation ? 'Create' : 'Update'}
          </Button>
        </Flex>
      </Card>

      <Webhooks webhookReceivers={webhookReceivers} className='mt-5' />
    </Flex>
  );
};
