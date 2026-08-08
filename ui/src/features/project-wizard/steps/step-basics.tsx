import { zodResolver } from '@hookform/resolvers/zod';
import { Card, Input } from 'antd';
import { useEffect, useRef } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { FieldContainer } from '@ui/features/common/form/field-container';
import { validatorMessages } from '@ui/utils/validators';

import { BasicsState, projectNameRegex } from '../types';

const formSchema = z.object({
  name: z
    .string()
    .min(1, { error: validatorMessages.required })
    .max(63, { error: 'Must be at most 63 characters' })
    .regex(projectNameRegex, {
      error: 'Must be a valid DNS-1123 name: lowercase letters, numbers, and dashes'
    }),
  description: z.string()
});

type StepBasicsProps = {
  value: BasicsState;
  onChange: (value: BasicsState) => void;
};

export const StepBasics = ({ value, onChange }: StepBasicsProps) => {
  const { control, watch } = useForm<BasicsState>({
    defaultValues: value,
    resolver: zodResolver(formSchema),
    mode: 'onChange'
  });

  // Push every edit up into the wizard draft without re-subscribing when the
  // parent recreates its callback.
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  useEffect(() => {
    const subscription = watch((formValues) => onChangeRef.current(formValues as BasicsState));
    return () => subscription.unsubscribe();
  }, [watch]);

  return (
    <div className='flex flex-col gap-4'>
      <Card title='Identity' extra={<span className='text-xs text-gray-400'>kind: Project</span>}>
        <FieldContainer
          control={control}
          name='name'
          label='Project name'
          required
          description='Lowercase letters, numbers, and dashes. Used as the Namespace name.'
        >
          {({ field }) => (
            <Input
              {...field}
              className='font-mono'
              placeholder='my-app-delivery'
              autoComplete='off'
            />
          )}
        </FieldContainer>
        <FieldContainer
          control={control}
          name='description'
          label='Description'
          description='Surfaces on the project list and project home.'
        >
          {({ field }) => (
            <Input.TextArea
              {...field}
              rows={2}
              placeholder='GitOps promotion for the checkout service across test → uat → prod.'
            />
          )}
        </FieldContainer>
      </Card>
    </div>
  );
};
