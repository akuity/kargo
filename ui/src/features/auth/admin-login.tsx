import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { Button, Input } from 'antd';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { adminLogin } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { zodValidators } from '@ui/utils/validators';

import { FieldContainer } from '../common/form/field-container';

import { useAuth } from './use-auth';

const formSchema = z.object({
  password: zodValidators.requiredString
});

export const AdminLogin = () => {
  const { onLogin } = useAuth();
  const { mutate, isLoading } = useMutation({
    ...adminLogin.useMutation(),
    onSuccess: (response) => onLogin(response.idToken)
  });

  const { control, handleSubmit } = useForm({
    defaultValues: {
      password: ''
    },
    resolver: zodResolver(formSchema)
  });

  const onSubmit = handleSubmit((values) => {
    mutate(values);
  });

  return (
    <form onSubmit={onSubmit}>
      <FieldContainer label='Password' name='password' control={control}>
        {({ field }) => <Input {...field} type='password' />}
      </FieldContainer>
      <Button htmlType='submit' loading={isLoading} type='primary' block className='mt-2'>
        Login
      </Button>
    </form>
  );
};
