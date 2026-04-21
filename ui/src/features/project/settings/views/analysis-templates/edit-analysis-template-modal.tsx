import { useQueryClient } from '@tanstack/react-query';
import { Modal } from 'antd';
import { useForm } from 'react-hook-form';
import { stringify } from 'yaml';

import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalProps } from '@ui/features/common/modal/use-modal';
import { useUpdateResource } from '@ui/gen/api/v2/resources/resources';
import {
  getListAnalysisTemplatesQueryKey,
  useGetAnalysisTemplate
} from '@ui/gen/api/v2/verifications/verifications';

import { getAnalysisTemplateYAMLExample } from './utils/analysis-template-example';

type Props = ModalProps & {
  templateName: string;
  projectName: string;
};

export const EditAnalysisTemplateModal = ({ visible, hide, templateName, projectName }: Props) => {
  const queryClient = useQueryClient();

  const { mutate, isPending } = useUpdateResource({
    mutation: {
      onSuccess: () => {
        queryClient.invalidateQueries({
          queryKey: getListAnalysisTemplatesQueryKey(projectName)
        });
        hide();
      }
    }
  });

  const { data: templateResponse, isLoading } = useGetAnalysisTemplate(projectName, templateName);

  const { control, handleSubmit } = useForm({
    values: {
      value: templateResponse?.data ? stringify(templateResponse.data) : ''
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
            placeholder={getAnalysisTemplateYAMLExample(projectName)}
            isLoading={isLoading}
            label='Spec'
          />
        )}
      </FieldContainer>
    </Modal>
  );
};
