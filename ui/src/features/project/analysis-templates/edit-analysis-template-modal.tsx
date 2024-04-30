import { createConnectQueryKey, useMutation, useQuery } from '@connectrpc/connect-query';
import { useQueryClient } from '@tanstack/react-query';
import { Modal } from 'antd';
import { useForm } from 'react-hook-form';

import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalProps } from '@ui/features/common/modal/use-modal';
import {
  getAnalysisTemplate,
  listAnalysisTemplates,
  updateResource
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { RawFormat } from '@ui/gen/service/v1alpha1/service_pb';
import { decodeRawData } from '@ui/utils/decode-raw-data';

import { getAnalysisTemplateYAMLExample } from './utils/analysis-template-example';

type Props = ModalProps & {
  templateName: string;
  projectName: string;
};

export const EditAnalysisTemplateModal = ({ visible, hide, templateName, projectName }: Props) => {
  const queryClient = useQueryClient();

  const { mutateAsync, isPending } = useMutation(updateResource, {
    onSuccess: () => hide()
  });

  const { data: templateResponse, isLoading } = useQuery(getAnalysisTemplate, {
    project: projectName,
    name: templateName,
    format: RawFormat.YAML
  });

  const { control, handleSubmit } = useForm({
    values: {
      value: decodeRawData(templateResponse)
    }
  });

  const onSubmit = handleSubmit(async (data) => {
    const textEncoder = new TextEncoder();

    await mutateAsync(
      {
        manifest: textEncoder.encode(data.value)
      },
      {
        onSuccess: () =>
          queryClient.invalidateQueries({ queryKey: createConnectQueryKey(listAnalysisTemplates) })
      }
    );
  });

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
            isHideManagedFieldsDisplayed
            label='Spec'
          />
        )}
      </FieldContainer>
    </Modal>
  );
};
