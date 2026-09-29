import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Flex, message, notification } from 'antd';
import Card from 'antd/es/card/Card';
import { JSONSchema4 } from 'json-schema';
import { useMemo } from 'react';
import { useForm } from 'react-hook-form';
import { stringify } from 'yaml';
import { z } from 'zod';

import { YamlEditor } from '@ui/features/common/code-editor/yaml-editor';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { useCreateResource, useUpdateResource } from '@ui/gen/api/v2/resources/resources';
import { useGetClusterConfig } from '@ui/gen/api/v2/system/system';
import clusterConfigSchema from '@ui/gen/schema/clusterconfigs.kargo.akuity.io_v1alpha1.json';
import { zodValidators } from '@ui/utils/validators';

import { clusterConfigYAMLExample } from './cluster-config-yaml-example';
import { Refresh } from './refresh';

const formSchema = z.object({
  value: zodValidators.requiredString
});

export const ClusterConfig = () => {
  const getClusterConfigQuery = useGetClusterConfig({
    query: { meta: { silent404: true } }
  });

  const clusterConfigObject = getClusterConfigQuery.data?.data;
  const clusterConfigYAML = useMemo(() => {
    if (!clusterConfigObject) {
      return '';
    }
    try {
      return stringify(clusterConfigObject);
    } catch (e) {
      notification.error({
        message: (e as Error)?.message || 'Failed to stringify ClusterConfig',
        placement: 'bottomRight'
      });
      return '';
    }
  }, [clusterConfigObject]);

  const creation = !clusterConfigYAML;

  const clusterConfigForm = useForm({
    values: {
      value: clusterConfigYAML
    },
    resolver: zodResolver(formSchema)
  });

  const mutationOptions = {
    mutation: {
      onSuccess: () => {
        message.success({
          content: `ClusterConfig has been ${creation ? 'created' : 'updated'}`
        });
        getClusterConfigQuery.refetch();
      }
    }
  };

  const createMutation = useCreateResource(mutationOptions);
  const updateMutation = useUpdateResource(mutationOptions);

  const createOrUpdateMutation = creation ? createMutation : updateMutation;

  const onSubmitConfig = clusterConfigForm.handleSubmit((data) =>
    createOrUpdateMutation.mutate({ data: data.value })
  );

  return (
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
          />
        )}
      </FieldContainer>
      <Flex justify='end'>
        <Button type='primary' onClick={onSubmitConfig} loading={createOrUpdateMutation.isPending}>
          {creation ? 'Create' : 'Update'}
        </Button>
      </Flex>
    </Card>
  );
};
