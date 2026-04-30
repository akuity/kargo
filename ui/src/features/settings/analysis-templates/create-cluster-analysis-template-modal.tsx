import { useQueryClient } from '@tanstack/react-query';
import { Modal } from 'antd';
import { useForm } from 'react-hook-form';

import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalProps } from '@ui/features/common/modal/use-modal';
import { getClusterAnalysisTemplateYAMLExample } from '@ui/features/utils/cluster-analysis-template-example';
import { useCreateResource } from '@ui/gen/api/v2/resources/resources';
import { getListClusterAnalysisTemplatesQueryKey } from '@ui/gen/api/v2/verifications/verifications';

export const CreateClusterAnalysisTemplateModal = ({ visible, hide }: ModalProps) => {
  const queryClient = useQueryClient();

  const { mutate, isPending } = useCreateResource({
    mutation: {
      onSuccess: () => {
        queryClient.invalidateQueries({
          queryKey: getListClusterAnalysisTemplatesQueryKey()
        });
        hide();
      }
    }
  });

  const { control, handleSubmit } = useForm({
    defaultValues: {
      value: getClusterAnalysisTemplateYAMLExample()
    }
  });

  const onSubmit = handleSubmit((data) => mutate({ data: data.value }));

  return (
    <Modal
      open={visible}
      onCancel={hide}
      title='Create Cluster Analysis Template'
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
            placeholder={getClusterAnalysisTemplateYAMLExample()}
          />
        )}
      </FieldContainer>
    </Modal>
  );
};
