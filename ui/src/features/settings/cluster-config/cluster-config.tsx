import { useMutation, useQuery } from '@connectrpc/connect-query';
import { faPlus } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Flex, message, notification } from 'antd';
import Card from 'antd/es/card/Card';
import { JSONSchema4 } from 'json-schema';
import { useMemo } from 'react';
import { useForm } from 'react-hook-form';
import { useParams } from 'react-router-dom';
import { parse, stringify } from 'yaml';
import { z } from 'zod';

import { YamlEditor } from '@ui/features/common/code-editor/yaml-editor';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { useModal } from '@ui/features/common/modal/use-modal';
import { Webhooks } from '@ui/features/project/settings/views/project-config/webhooks';
import {
  createOrUpdateResource,
  getClusterConfig
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { RawFormat } from '@ui/gen/api/service/v1alpha1/service_pb';
import { ClusterConfig as ClusterConfigT } from '@ui/gen/api/v1alpha1/generated_pb';
import clusterConfigSchema from '@ui/gen/schema/clusterconfigs.kargo.akuity.io_v1alpha1.json';
import { decodeRawData } from '@ui/utils/decode-raw-data';
import { zodValidators } from '@ui/utils/validators';

import { clusterConfigYAMLExample } from './cluster-config-yaml-example';
import { Refresh } from './refresh';
import { clusterConfigTransport } from './transport';
import { CreateWebhookModal } from './webhook/create-webhook-modal';

const formSchema = z.object({
  value: zodValidators.requiredString
});

export const ClusterConfig = () => {
  const { name } = useParams();

  const getClusterConfigQuery = useQuery(
    getClusterConfig,
    { format: RawFormat.YAML },
    { transport: clusterConfigTransport }
  );

  const clusterConfigYAML = decodeRawData(getClusterConfigQuery.data);

  const clusterConfig = useMemo(() => {
    try {
      return parse(clusterConfigYAML) as ClusterConfigT;
    } catch (e) {
      notification.error({
        message: (e as Error)?.message || 'Failed to parse ClusterConfig YAML',
        placement: 'bottomRight'
      });
    }
  }, [clusterConfigYAML]);

  const webhookReceivers = clusterConfig?.status?.webhookReceivers || [];

  const creation = !clusterConfigYAML;

  const clusterConfigForm = useForm({
    values: {
      value: clusterConfigYAML
    },
    resolver: zodResolver(formSchema)
  });

  const createOrUpdateMutation = useMutation(createOrUpdateResource, {
    onSuccess: () => {
      message.success({
        content: `ClusterConfig has been ${creation ? 'created' : 'updated'}`
      });
      getClusterConfigQuery.refetch();
    }
  });

  const createWebhookModal = useModal((props) => (
    <CreateWebhookModal clusterConfigYAML={clusterConfigYAML} project={name || ''} {...props} />
  ));

  const onSubmitConfig = clusterConfigForm.handleSubmit(async (data) => {
    const textEncoder = new TextEncoder();

    await createOrUpdateMutation.mutateAsync({
      manifest: textEncoder.encode(data.value)
    });
  });

  return (
    <Flex gap={16} vertical>
      <Card title='Cluster Config' type='inner' extra={clusterConfigYAML !== '' && <Refresh />}>
        <FieldContainer control={clusterConfigForm.control} name='value'>
          {({ field }) => (
            <YamlEditor
              label='YAML'
              isLoading={getClusterConfigQuery.isLoading}
              height='500px'
              value={field.value}
              onChange={(e) => field.onChange(e || '')}
              placeholder={stringify(clusterConfigYAMLExample)}
              schema={clusterConfigSchema as JSONSchema4}
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

      <Webhooks webhookReceivers={webhookReceivers} className='mt-5' />
    </Flex>
  );
};
