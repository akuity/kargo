import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Card, Flex, message, notification } from 'antd';
import type { JSONSchema4 } from 'json-schema';
import { useMemo } from 'react';
import { useForm } from 'react-hook-form';
import { useParams } from 'react-router-dom';
import yaml, { stringify } from 'yaml';
import { z } from 'zod';

import { YamlEditor } from '@ui/features/common/code-editor/yaml-editor';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { projectConfigYAMLExample } from '@ui/features/project/list/utils/project-yaml-example';
import { useGetProjectConfig } from '@ui/gen/api/v2/core/core';
import { useCreateResource, useUpdateResource } from '@ui/gen/api/v2/resources/resources';
import projectConfigSchema from '@ui/gen/schema/projectconfigs.kargo.akuity.io_v1alpha1.json';
import { zodValidators } from '@ui/utils/validators';

import { Refresh } from './refresh';

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

  const creation = !projectConfigYAML;

  const projectConfigForm = useForm({
    values: {
      value: projectConfigYAML
    },
    resolver: zodResolver(formSchema)
  });

  const mutationOptions = {
    mutation: {
      onSuccess: () => {
        message.success({ content: `ProjectConfig has been ${creation ? 'created' : 'updated'}` });
        projectConfigQuery.refetch();
      }
    }
  };

  const createMutation = useCreateResource(mutationOptions);
  const updateMutation = useUpdateResource(mutationOptions);

  const createOrUpdateMutation = creation ? createMutation : updateMutation;

  const onSubmitConfig = projectConfigForm.handleSubmit((data) =>
    createOrUpdateMutation.mutate({ data: data.value })
  );

  return (
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
      <Flex justify='end'>
        <Button type='primary' onClick={onSubmitConfig} loading={createOrUpdateMutation.isPending}>
          {creation ? 'Create' : 'Update'}
        </Button>
      </Flex>
    </Card>
  );
};
