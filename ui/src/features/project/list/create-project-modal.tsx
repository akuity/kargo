import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { Modal } from 'antd';
import type { JSONSchema4 } from 'json-schema';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { YamlEditor } from '@ui/features/common/code-editor/yaml-editor';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import schema from '@ui/gen/schema/projects.kargo.akuity.io_v1alpha1.json';
import { createResource } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { zodValidators } from '@ui/utils/validators';

import { projectYAMLExample } from './utils/project-yaml-example';

const formSchema = z.object({
  value: zodValidators.requiredString
});

export const CreateProjectModal = ({ visible, hide }: ModalComponentProps) => {
  const { mutateAsync, isPending } = useMutation({
    ...createResource.useMutation(),
    onSuccess: () => hide()
  });

  const { control, handleSubmit } = useForm({
    defaultValues: {
      value: projectYAMLExample
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
      title='Create Project'
      width={680}
      onCancel={hide}
      okText='Create'
      onOk={onSubmit}
      okButtonProps={{ loading: isPending }}
    >
      <FieldContainer label='YAML' name='value' control={control}>
        {({ field: { value, onChange } }) => (
          <YamlEditor
            value={value}
            onChange={(e) => onChange(e || '')}
            height='500px'
            schema={schema as JSONSchema4}
            placeholder={projectYAMLExample}
          />
        )}
      </FieldContainer>
    </Modal>
  );
};
