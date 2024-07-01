import { useMutation } from '@connectrpc/connect-query';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Modal, Space, Typography } from 'antd';
import type { JSONSchema4 } from 'json-schema';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { YamlEditor } from '@ui/features/common/code-editor/yaml-editor';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import schema from '@ui/gen/schema/stages.kargo.akuity.io_v1alpha1.json';
import { createResource } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { zodValidators } from '@ui/utils/validators';

import { getStageYAMLExample } from './utils/stage-yaml-example';

type Props = ModalComponentProps & {
  project: string;
};

const formSchema = z.object({
  value: zodValidators.requiredString
});

export const CreateStageModal = ({ visible, hide, project }: Props) => {
  const { mutateAsync, isPending } = useMutation(createResource, {
    onSuccess: () => hide()
  });

  const { control, handleSubmit } = useForm({
    defaultValues: {
      value: getStageYAMLExample(project)
    },
    resolver: zodResolver(formSchema)
  });

  const onSubmit = handleSubmit(async (data) => {
    const textEncoder = new TextEncoder();
    await mutateAsync({
      manifest: textEncoder.encode(data.value)
    });
  });

  return (
    <Modal
      open={visible}
      title='Create Stage'
      closable={false}
      width={680}
      footer={
        <div className='flex items-center justify-between'>
          <Typography.Link
            href='https://kargo.akuity.io/quickstart/#the-test-stage'
            target='_blank'
          >
            Documentation
          </Typography.Link>
          <Space>
            <Button onClick={hide}>Cancel</Button>
            <Button type='primary' onClick={onSubmit} loading={isPending}>
              Create
            </Button>
          </Space>
        </div>
      }
    >
      <FieldContainer label='YAML' name='value' control={control}>
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
    </Modal>
  );
};

export default CreateStageModal;
