import { createConnectQueryKey, useMutation } from '@connectrpc/connect-query';
import { useQueryClient } from '@tanstack/react-query';
import { Button, Flex, Form, Input } from 'antd';
import React from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import yaml from 'yaml';

import { paths } from '@ui/config/paths';
import {
  deleteResource,
  listProjects
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

import { projectYAMLExample } from '../../../list/utils/project-yaml-example';

export const DeleteProject = () => {
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
    <>
      <Form layout='vertical' component='div'>
        <Form.Item label='Confirm Project name' className='max-w-sm'>
          <Flex gap={16}>
            <Input
              onChange={(e) => setInputValue(e.target.value)}
              value={inputValue}
              placeholder={name}
            />
            <Button
              type='primary'
              onClick={onDelete}
              loading={isPending}
              danger
              disabled={name !== inputValue}
            >
              Delete
            </Button>
          </Flex>
        </Form.Item>
      </Form>
    </>
  );
};
