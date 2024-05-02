import { faPlus } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button } from 'antd';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { FieldContainer } from '@ui/features/common/form/field-container';
import { MultiStringEditor } from '@ui/features/common/form/multi-string-editor';
import { PolicyRule } from '@ui/gen/k8s.io/api/rbac/v1/generated_pb';

const ruleFormSchema = () =>
  z.object({
    verbs: z.string().array(),
    apiGroups: z.string().array(),
    resources: z.string().array(),
    resourceNames: z.string().array(),
    nonResourceURLs: z.string().array()
  });

export const RuleEditor = ({ onSuccess }: { onSuccess: (rule: PolicyRule) => void }) => {
  const { control, handleSubmit, reset } = useForm({
    resolver: zodResolver(ruleFormSchema())
  });

  const onSubmit = handleSubmit((values) => {
    onSuccess({
      verbs: values.verbs,
      apiGroups: values.apiGroups,
      resources: values.resources,
      resourceNames: values.resourceNames,
      nonResourceURLs: values.nonResourceURLs
    } as PolicyRule);
    reset({
      verbs: [] as string[],
      apiGroups: [] as string[],
      resources: [] as string[],
      resourceNames: [] as string[],
      nonResourceURLs: [] as string[]
    } as PolicyRule);
  });

  return (
    <div>
      <div className='mx-auto font-semibold mb-2 text-gray-500 text-center text-sm'>NEW RULE</div>

      <div className='rounded-md p-3 border-2 border-gray-100 border-solid'>
        <div className='flex gap-4'>
          <FieldContainer control={control} name='verbs' className='w-full'>
            {({ field }) => (
              <MultiStringEditor
                value={field.value}
                onChange={field.onChange}
                placeholder='get'
                label='Verbs'
              />
            )}
          </FieldContainer>

          <FieldContainer control={control} name='apiGroups' className='w-full'>
            {({ field }) => (
              <MultiStringEditor
                value={field.value}
                onChange={field.onChange}
                placeholder='core'
                label='API Groups'
              />
            )}
          </FieldContainer>

          <FieldContainer control={control} name='resources' className='w-full'>
            {({ field }) => (
              <MultiStringEditor
                value={field.value}
                onChange={field.onChange}
                placeholder='pods'
                label='Resources'
              />
            )}
          </FieldContainer>
        </div>

        <div className='flex gap-4'>
          <FieldContainer control={control} name='resourceNames' className='w-full'>
            {({ field }) => (
              <MultiStringEditor
                value={field.value}
                onChange={field.onChange}
                placeholder='my-pod'
                label='Resource Names'
              />
            )}
          </FieldContainer>

          <FieldContainer control={control} name='nonResourceURLs' className='w-full'>
            {({ field }) => (
              <MultiStringEditor
                value={field.value}
                onChange={field.onChange}
                placeholder='https://example.com'
                label='Non-Resource URLs'
              />
            )}
          </FieldContainer>

          <div className='flex items-end w-full'>
            <Button onClick={onSubmit} icon={<FontAwesomeIcon icon={faPlus} />} className='ml-auto'>
              Add Rule
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
};
