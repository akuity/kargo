import { createConnectQueryKey, useMutation } from '@connectrpc/connect-query';
import { useQueryClient } from '@tanstack/react-query';
import { Modal } from 'antd';
import { useForm } from 'react-hook-form';

import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalProps } from '@ui/features/common/modal/use-modal';
import { queryCache } from '@ui/features/utils/cache';
import {
  createResource,
  listAnalysisTemplates
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { decodeUint8ArrayYamlManifestToJson } from '@ui/utils/decode-raw-data';

import { getAnalysisTemplateYAMLExample } from './utils/analysis-template-example';

type Props = ModalProps & {
  namespace: string;
};

export const CreateAnalysisTemplateModal = ({ visible, hide, namespace }: Props) => {
  const queryClient = useQueryClient();

  const { mutateAsync, isPending } = useMutation(createResource, {
    onSuccess: (response) => {
      for (const result of response?.results || []) {
        if (result?.result?.case === 'createdResourceManifest') {
          queryCache.analysisTemplates.add(namespace || '', [
            decodeUint8ArrayYamlManifestToJson(result?.result?.value)
          ]);
        }
      }
      hide();
    }
  });

  const { control, handleSubmit } = useForm({
    defaultValues: {
      value: getAnalysisTemplateYAMLExample(namespace)
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
