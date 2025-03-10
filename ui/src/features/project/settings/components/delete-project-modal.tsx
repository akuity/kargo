import { createConnectQueryKey, useMutation } from '@connectrpc/connect-query';
import { useQueryClient } from '@tanstack/react-query';
import { Alert, Form, Input, Modal } from 'antd';
import React from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import yaml from 'yaml';

import { paths } from '@ui/config/paths';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import {
  deleteResource,
  listProjects
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

import { projectYAMLExample } from '../../list/utils/project-yaml-example';

export const DeleteProjectModal = ({ visible, hide }: ModalComponentProps) => {
  const { name } = useParams();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [inputValue, setInputValue] = React.useState('');

  const { mutate, isPending } = useMutation(deleteResource, {
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({ schema: listProjects, cardinality: 'finite' })
      });
      navigate(paths.projects);
    }
  });

  const onDelete = () => {
    const textEncoder = new TextEncoder();
    const manifest = {
      ...projectYAMLExample,
      metadata: {
        name
      }
    };

    mutate({ manifest: textEncoder.encode(yaml.stringify(manifest)) });
  };

  return (
    <Modal
      destroyOnClose
      open={visible}
      title='Danger Zone'
      onCancel={hide}
      onOk={onDelete}
      okText='Delete'
      okButtonProps={{ loading: isPending, danger: true, disabled: name !== inputValue }}
    >
      <Alert
        type='error'
        banner
        message='Are you sure you want to delete Project?'
        className='mb-4'
        showIcon={false}
      />
      <Form layout='vertical' component='div'>
        <Form.Item label='If yes, please type the project name below:'>
          <Input
            onChange={(e) => setInputValue(e.target.value)}
            value={inputValue}
            placeholder={name}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
};
