import { useQueryClient } from '@tanstack/react-query';
import { Modal } from 'antd';
import { useForm } from 'react-hook-form';
import { stringify } from 'yaml';

import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalProps } from '@ui/features/common/modal/use-modal';
import { getClusterAnalysisTemplateYAMLExample } from '@ui/features/utils/cluster-analysis-template-example';
import { useUpdateResource } from '@ui/gen/api/v2/resources/resources';
import {
  getListClusterAnalysisTemplatesQueryKey,
  useGetClusterAnalysisTemplate
} from '@ui/gen/api/v2/verifications/verifications';

type Props = ModalProps & {
  templateName: string;
};

export const EditClusterAnalysisTemplateModal = ({ visible, hide, templateName }: Props) => {
  const queryClient = useQueryClient();

  const { mutate, isPending } = useUpdateResource({
    mutation: {
      onSuccess: () => {
        queryClient.invalidateQueries({
          queryKey: getListClusterAnalysisTemplatesQueryKey()
        });
        hide();
      }
    }
  });

  const { data: templateResponse, isLoading } = useGetClusterAnalysisTemplate(templateName);

  const templateYAML = templateResponse?.data ? stringify(templateResponse.data) : '';

  const { control, handleSubmit } = useForm({
    values: {
      value: templateYAML
    }
  });

  const onSubmit = handleSubmit((data) => mutate({ data: data.value }));

  return (
    <Modal
      open={visible}
      onCancel={hide}
      title='Edit Analysis Template'
      okText='Update'
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
            isLoading={isLoading}
            label='Spec'
          />
        )}
      </FieldContainer>
    </Modal>
  );
};
