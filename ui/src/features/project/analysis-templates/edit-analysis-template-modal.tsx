import { useMutation } from '@connectrpc/connect-query';
import { Modal } from 'antd';
import { useForm } from 'react-hook-form';
import yaml from 'yaml';

import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalProps } from '@ui/features/common/modal/use-modal';
import { AnalysisTemplate } from '@ui/gen/api/v1alpha1/generated_pb';
import { updateResource } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

import { getAnalysisTemplateYAMLExample } from './utils/analysis-template-example';

type Props = ModalProps & {
  template: AnalysisTemplate;
};

export const EditAnalysisTemplateModal = ({ visible, hide, template }: Props) => {
  const { mutateAsync, isPending } = useMutation(updateResource, {
    onSuccess: () => hide()
  });

  const { control, handleSubmit } = useForm({
    defaultValues: {
      value: yaml.stringify(template)
    }
  });

  const onSubmit = handleSubmit(async (data) => {
    const textEncoder = new TextEncoder();

    await mutateAsync({
      manifest: textEncoder.encode(
        yaml.stringify({
          apiVersion: 'argoproj.io/v1alpha1',
          kind: 'AnalysisTemplate',
          metadata: template.metadata,
          spec: data
        })
      )
    });
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
      <FieldContainer name='value' control={control} label='Spec'>
        {({ field: { value, onChange } }) => (
          <YamlEditor
            value={value}
            onChange={(e) => onChange(e || '')}
            height='500px'
            placeholder={getAnalysisTemplateYAMLExample(template.metadata?.namespace || '')}
          />
        )}
      </FieldContainer>
    </Modal>
  );
};
