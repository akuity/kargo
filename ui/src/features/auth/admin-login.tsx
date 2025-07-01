import { useMutation } from '@connectrpc/connect-query';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Input } from 'antd';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { adminLogin } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { zodValidators } from '@ui/utils/validators';

import { FieldContainer } from '../common/form/field-container';

import { useAuthContext } from './context/use-auth-context';

const formSchema = z.object({
  password: zodValidators.requiredString
});

export const AdminLogin = () => {
  const { login } = useAuthContext();
  const { mutate, isPending } = useMutation(adminLogin, {
    onSuccess: (response) => login(response.idToken)
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
      <Button htmlType='submit' loading={isPending} type='primary' block className='mt-2'>
        Login
      </Button>
    </form>
  );
};
