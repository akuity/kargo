import { useQueryClient } from '@tanstack/react-query';
import { Button, Flex, Form, Input } from 'antd';
import React from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { getListProjectsQueryKey, useDeleteProject } from '@ui/gen/api/v2/core/core';

export const DeleteProject = () => {
  const { name } = useParams();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [inputValue, setInputValue] = React.useState('');

  const { mutate, isPending } = useDeleteProject({
    mutation: {
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: getListProjectsQueryKey() });
        navigate(paths.projects);
      }
    }
  });

  const onDelete = () => {
    mutate({ project: name || '' });
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
