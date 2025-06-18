import { useMutation, useQuery } from '@connectrpc/connect-query';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Flex, message } from 'antd';
import Card from 'antd/es/card/Card';
import { JSONSchema4 } from 'json-schema';
import { useForm } from 'react-hook-form';
import { stringify } from 'yaml';
import { z } from 'zod';

import { YamlEditor } from '@ui/features/common/code-editor/yaml-editor';
import { FieldContainer } from '@ui/features/common/form/field-container';
import {
  createOrUpdateResource,
  getClusterConfig
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { RawFormat } from '@ui/gen/api/service/v1alpha1/service_pb';
import clusterConfigSchema from '@ui/gen/schema/clusterconfigs.kargo.akuity.io_v1alpha1.json';
import { decodeRawData } from '@ui/utils/decode-raw-data';
import { zodValidators } from '@ui/utils/validators';

import { clusterConfigYAMLExample } from './cluster-config-yaml-example';
import { clusterConfigTransport } from './transport';

const formSchema = z.object({
  value: zodValidators.requiredString
});

export const ClusterConfig = () => {
  const getClusterConfigQuery = useQuery(
    getClusterConfig,
    { format: RawFormat.YAML },
    { transport: clusterConfigTransport }
  );

  const clusterConfigYAML = decodeRawData(getClusterConfigQuery.data);

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

  const onSubmitConfig = clusterConfigForm.handleSubmit(async (data) => {
    const textEncoder = new TextEncoder();

    await createOrUpdateMutation.mutateAsync({
      manifest: textEncoder.encode(data.value)
    });
  });

  return (
    <Card title='Cluster Config' type='inner'>
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
      <Flex justify='flex-end'>
        <Button type='primary' onClick={onSubmitConfig} loading={createOrUpdateMutation.isPending}>
          {creation ? 'Create' : 'Update'}
        </Button>
      </Flex>
    </Card>
  );
};
