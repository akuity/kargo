import { createConnectQueryKey, useMutation } from '@connectrpc/connect-query';
import { useQueryClient } from '@tanstack/react-query';
import { Modal } from 'antd';
import { useForm } from 'react-hook-form';

import { transportWithAuth } from '@ui/config/transport';
import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalProps } from '@ui/features/common/modal/use-modal';
import {
  createResource,
  listClusterAnalysisTemplates
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

import { getClusterAnalysisTemplateYAMLExample } from '../../utils/cluster-analysis-template-example';

export const CreateClusterAnalysisTemplateModal = ({ visible, hide }: ModalProps) => {
  const queryClient = useQueryClient();

  const { mutateAsync, isPending } = useMutation(createResource, {
    onSuccess: () => hide()
  });

  const { control, handleSubmit } = useForm({
    defaultValues: {
      value: getClusterAnalysisTemplateYAMLExample()
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
          queryClient.invalidateQueries({
            queryKey: createConnectQueryKey({
              schema: listClusterAnalysisTemplates,
              cardinality: 'finite',
              transport: transportWithAuth
            })
          })
      }
    );
  });

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
