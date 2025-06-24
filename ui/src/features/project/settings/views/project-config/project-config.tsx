import { useMutation, useQuery } from '@connectrpc/connect-query';
import { faPlus } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Card, Flex, message } from 'antd';
import type { JSONSchema4 } from 'json-schema';
import { useForm } from 'react-hook-form';
import { useParams } from 'react-router-dom';
import yaml from 'yaml';
import { z } from 'zod';

import { YamlEditor } from '@ui/features/common/code-editor/yaml-editor';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { useModal } from '@ui/features/common/modal/use-modal';
import { projectConfigYAMLExample } from '@ui/features/project/list/utils/project-yaml-example';
import {
  createOrUpdateResource,
  getProjectConfig
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { RawFormat } from '@ui/gen/api/service/v1alpha1/service_pb';
import projectConfigSchema from '@ui/gen/schema/projectconfigs.kargo.akuity.io_v1alpha1.json';
import { decodeRawData } from '@ui/utils/decode-raw-data';
import { zodValidators } from '@ui/utils/validators';

import { Refresh } from './refresh';
import { projectConfigTransport } from './transport';
import { CreateWebhookModal } from './webhook/create-webhook-modal';
import { Webhooks } from './webhooks';

const formSchema = z.object({
  value: zodValidators.requiredString
});

export const ProjectConfig = () => {
  const { name } = useParams();

  const projectConfigQuery = useQuery(
    getProjectConfig,
    { project: name, format: RawFormat.YAML },
    {
      transport: projectConfigTransport
    }
  );

  const projectConfigYAML = decodeRawData(projectConfigQuery.data);

  const creation = !projectConfigYAML;

  const projectConfigForm = useForm({
    values: {
      value: projectConfigYAML
    },
    resolver: zodResolver(formSchema)
  });

  const createOrUpdateMutation = useMutation(createOrUpdateResource, {
    onSuccess: () => {
      message.success({ content: `ProjectConfig has been ${creation ? 'created' : 'updated'}` });
      projectConfigQuery.refetch();
    }
  });

  const onSubmitConfig = projectConfigForm.handleSubmit(async (data) => {
    const textEncoder = new TextEncoder();
    await createOrUpdateMutation.mutateAsync({
      manifest: textEncoder.encode(data.value)
    });
  });

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
              isHideManagedFieldsDisplayed
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

      <Webhooks projectConfigYAML={projectConfigYAML} className='mt-5' />
    </Flex>
  );
};
