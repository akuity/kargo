import { createConnectQueryKey, useMutation } from '@connectrpc/connect-query';
import { zodResolver } from '@hookform/resolvers/zod';
import { Input, Modal } from 'antd';
import { useForm } from 'react-hook-form';

import { queryClient } from '@ui/config/query-client';
import {
  listConfigMaps,
  updateConfigMap
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { ConfigMap } from '@ui/gen/k8s.io/api/core/v1/generated_pb';

import { FieldContainer } from '../../form/field-container';
import { ModalProps } from '../../modal/use-modal';
import { ObjectEditor } from '../../object-editor';

import { confgMapSchema } from './schema';

type Props = ModalProps & {
  project: string;
  configMap: ConfigMap;
};

export const EditConfigMapModal = ({ configMap, project, hide, visible }: Props) => {
  const { control, handleSubmit } = useForm({
    defaultValues: {
      name: configMap.metadata?.name,
      data: { ...configMap.data }
    },
    resolver: zodResolver(confgMapSchema)
  });

  const { mutate, isPending } = useMutation(updateConfigMap, {
    onSuccess: () => {
      queryClient.refetchQueries({
        queryKey: createConnectQueryKey({
          schema: listConfigMaps,
          cardinality: 'finite'
        })
      });
      hide();
    }
  });

  const onSubmit = handleSubmit((data) => mutate({ ...data, project }));

  return (
    <Modal
      okText='Update'
      onOk={onSubmit}
      onCancel={hide}
      open={visible}
      title='Edit ConfigMap'
      okButtonProps={{ loading: isPending }}
    >
      <FieldContainer label='Name' name='name' control={control}>
        {({ field }) => <Input {...field} disabled />}
      </FieldContainer>
      <FieldContainer label='Data' name='data' control={control}>
        {({ field }) => <ObjectEditor value={field.value} onChange={field.onChange} />}
      </FieldContainer>
    </Modal>
  );
};
