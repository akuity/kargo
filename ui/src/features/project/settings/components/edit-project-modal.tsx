import { useMutation, useQuery } from '@connectrpc/connect-query';
import { zodResolver } from '@hookform/resolvers/zod';
import { Modal } from 'antd';
import type { JSONSchema4 } from 'json-schema';
import { useForm } from 'react-hook-form';
import { useParams } from 'react-router-dom';
import yaml from 'yaml';
import { z } from 'zod';

import { YamlEditor } from '@ui/features/common/code-editor/yaml-editor';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import {
  getProject,
  updateResource
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { RawFormat } from '@ui/gen/api/service/v1alpha1/service_pb';
import schema from '@ui/gen/schema/projects.kargo.akuity.io_v1alpha1.json';
import { decodeRawData } from '@ui/utils/decode-raw-data';
import { zodValidators } from '@ui/utils/validators';

import { projectYAMLExample } from '../../list/utils/project-yaml-example';

const formSchema = z.object({
  value: zodValidators.requiredString
});

export const EditProjectModal = ({ visible, hide }: ModalComponentProps) => {
  const { name } = useParams();
  const { data, isLoading } = useQuery(getProject, { name, format: RawFormat.YAML });

  const { mutateAsync, isPending } = useMutation(updateResource, {
    onSuccess: () => hide()
  });

  const { control, handleSubmit } = useForm({
    values: {
      value: decodeRawData(data)
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
      destroyOnClose
      open={visible}
      title='Edit Project'
      width={680}
      onCancel={hide}
      onOk={onSubmit}
      okText='Update'
      okButtonProps={{ loading: isPending }}
    >
      <FieldContainer name='value' control={control}>
        {({ field: { value, onChange } }) => (
          <YamlEditor
            label='YAML'
            value={value}
            onChange={(e) => onChange(e || '')}
            height='500px'
            schema={schema as JSONSchema4}
            placeholder={yaml.stringify(projectYAMLExample)}
            isLoading={isLoading}
            resourceType='projects'
            isHideManagedFieldsDisplayed
          />
        )}
      </FieldContainer>
    </Modal>
  );
};
