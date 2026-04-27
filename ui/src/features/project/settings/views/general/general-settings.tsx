import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Card, Flex, message } from 'antd';
import type { JSONSchema4 } from 'json-schema';
import { useForm } from 'react-hook-form';
import { useParams } from 'react-router-dom';
import yaml, { stringify } from 'yaml';
import { z } from 'zod';

import { YamlEditor } from '@ui/features/common/code-editor/yaml-editor';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { projectYAMLExample } from '@ui/features/project/list/utils/project-yaml-example';
import { useGetProject } from '@ui/gen/api/v2/core/core';
import { useUpdateResource } from '@ui/gen/api/v2/resources/resources';
import schema from '@ui/gen/schema/projects.kargo.akuity.io_v1alpha1.json';
import { zodValidators } from '@ui/utils/validators';

const formSchema = z.object({
  value: zodValidators.requiredString
});

import { DeleteProject } from './delete-project-modal';

export const GeneralSettings = () => {
  const { name } = useParams();
  const { data, isLoading } = useGetProject(name || '');

  const { mutate, isPending } = useUpdateResource({
    mutation: {
      onSuccess: () =>
        message.success({
          content: `Project Configuration has been updated.`
        })
    }
  });

  const { control, handleSubmit } = useForm({
    values: {
      value: data?.data ? stringify(data.data) : ''
    },
    resolver: zodResolver(formSchema)
  });

  const onSubmit = handleSubmit((data) => mutate({ data: data.value }));

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
            />
          )}
        </FieldContainer>
        <Flex justify='flex-end'>
          <Button type='primary' onClick={onSubmit} loading={isPending}>
            Update
          </Button>
        </Flex>
      </Card>

      <Card title='Delete Project' type='inner'>
        <DeleteProject />
      </Card>
    </Flex>
  );
};
