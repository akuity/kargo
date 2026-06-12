import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Flex, message, Space, Typography } from 'antd';
import type { JSONSchema4 } from 'json-schema';
import { useForm } from 'react-hook-form';
import { useParams } from 'react-router-dom';
import { stringify } from 'yaml';
import { z } from 'zod';

import { YamlEditor } from '@ui/features/common/code-editor/yaml-editor';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { useGetStage } from '@ui/gen/api/v2/core/core';
import { useUpdateResource } from '@ui/gen/api/v2/resources/resources';
import schema from '@ui/gen/schema/stages.kargo.akuity.io_v1alpha1.json';
import { zodValidators } from '@ui/utils/validators';

import { getStageYAMLExample } from '../../get-stage-yaml-example';

const formSchema = z.object({
  value: zodValidators.requiredString
});

export const StageEditForm = () => {
  const { name: projectName, stageName } = useParams();

  const getStageQuery = useGetStage(projectName || '', stageName || '');

  const { mutateAsync, isPending } = useUpdateResource({
    mutation: {
      onSuccess: () => message.success('Stage has been updated')
    }
  });

  const { control, handleSubmit } = useForm({
    values: {
      value: stringify(getStageQuery.data?.data)
    },
    resolver: zodResolver(formSchema)
  });

  const onSubmit = handleSubmit(async (data) => {
    await mutateAsync({
      data: data.value
    });
  });

  return (
    <>
      <FieldContainer name='value' control={control}>
        {({ field: { value, onChange } }) => (
          <YamlEditor
            value={value}
            onChange={(e) => onChange(e || '')}
            height='500px'
            schema={schema as JSONSchema4}
            placeholder={projectName && getStageYAMLExample(projectName)}
            isLoading={getStageQuery.isLoading}
            label='YAML'
            resourceType='stages'
          />
        )}
      </FieldContainer>

      <Flex justify='space-between'>
        <Typography.Link href='https://docs.kargo.io/quickstart/#the-test-stage' target='_blank'>
          Documentation
        </Typography.Link>
        <Space>
          <Button type='primary' onClick={onSubmit} loading={isPending}>
            Update
          </Button>
        </Space>
      </Flex>
    </>
  );
};
