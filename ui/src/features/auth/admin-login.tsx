import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { Button, Input } from 'antd';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { adminLogin } from '@ui/gen/api/v2/system/system';
import { zodValidators } from '@ui/utils/validators';

import { FieldContainer } from '../common/form/field-container';

import { useAuthContext } from './context/use-auth-context';

const formSchema = z.object({
  password: zodValidators.requiredString
});

export const AdminLogin = () => {
  const { login } = useAuthContext();
  const { mutate, isPending } = useMutation({
    mutationFn: (password: string) =>
      adminLogin({ headers: { Authorization: `Bearer ${password}` } }),
    onSuccess: (response) => {
      if (response.data?.idToken) {
        login(response.data.idToken);
      }
    }
  });

  const { control, handleSubmit } = useForm({
    defaultValues: {
      password: ''
    },
    resolver: zodResolver(formSchema)
  });

  const onSubmit = handleSubmit((values) => {
    mutate(values.password);
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
