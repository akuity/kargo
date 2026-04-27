import { useQueryClient } from '@tanstack/react-query';
import { Modal } from 'antd';
import { useForm } from 'react-hook-form';

import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalProps } from '@ui/features/common/modal/use-modal';
import { useCreateResource } from '@ui/gen/api/v2/resources/resources';
import { getListAnalysisTemplatesQueryKey } from '@ui/gen/api/v2/verifications/verifications';

import { getAnalysisTemplateYAMLExample } from './utils/analysis-template-example';

type Props = ModalProps & {
  namespace: string;
};

export const CreateAnalysisTemplateModal = ({ visible, hide, namespace }: Props) => {
  const queryClient = useQueryClient();

  const { mutate, isPending } = useCreateResource({
    mutation: {
      onSuccess: () => {
        queryClient.invalidateQueries({
          queryKey: getListAnalysisTemplatesQueryKey(namespace)
        });
        hide();
      }
    }
  });

  const { control, handleSubmit } = useForm({
    defaultValues: {
      value: getAnalysisTemplateYAMLExample(namespace)
    }
  });

  const onSubmit = handleSubmit((data) => mutate({ data: data.value }));

  return (
    <Modal
      open={visible}
      onCancel={hide}
      title='Create Analysis Template'
      okText='Create'
      onOk={onSubmit}
      okButtonProps={{ loading: isPending }}
      width={700}
    >
      <FieldContainer name='value' control={control}>
        {({ field: { value, onChange } }) => (
          <YamlEditor
            value={value}
            onChange={(e) => onChange(e || '')}
            height='500px'
            placeholder={getAnalysisTemplateYAMLExample(namespace)}
          />
        )}
      </FieldContainer>
    </Modal>
  );
};
