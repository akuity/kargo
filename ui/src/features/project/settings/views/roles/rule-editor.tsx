import { faPlus } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Select, SelectProps } from 'antd';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { FieldContainer } from '@ui/features/common/form/field-container';
import { MultiStringEditor } from '@ui/features/common/form/multi-string-editor';
import { PolicyRule } from '@ui/gen/k8s.io/api/rbac/v1/generated_pb';

const availableResources = [
  // core
  'events',
  'secrets',
  'serviceaccounts',
  // rbac.authorization.k8s.io
  'rolebindings',
  'roles',
  // kargo.akuity.io
  'freights',
  'stages',
  'promotions',
  'warehouses',
  // argoproj.io
  'analysisruns',
  'analysistemplates'
];

const availableVerbs = ['get', 'create', 'update', 'delete', 'patch', 'list', '*'];

const ruleFormSchema = () =>
  z.object({
    verbs: z.string().array(),
    resources: z.string().array(),
    resourceNames: z.string().array().optional()
  });

export const RuleEditor = ({
  onSuccess,
  style
}: {
  onSuccess: (rule: PolicyRule) => void;
  style?: React.CSSProperties;
}) => {
  const { control, handleSubmit, reset, watch } = useForm({
    resolver: zodResolver(ruleFormSchema())
  });

  const onSubmit = handleSubmit((values) => {
    onSuccess({
      verbs: values.verbs,
      resources: values.resources,
      resourceNames: values.resourceNames
    } as PolicyRule);
    reset({
      verbs: [] as string[],
      resources: [] as string[],
      resourceNames: [] as string[]
    } as PolicyRule);
  });

  const resources = watch('resources');

  const _Select = (props: SelectProps & { label: string }) => (
    <div>
      <div className='font-semibold text-xs text-gray-500 mb-2 mt-2'>{props.label}</div>
      <Select {...props} className='w-full' mode='tags' />
    </div>
  );

  return (
    <div style={style} className='-mt-7'>
      <div className='mx-auto font-semibold mb-2 text-gray-500 text-center text-sm'>NEW RULE</div>

      <div className='rounded-md p-3 border-2 border-gray-100 border-solid'>
        <div className='w-full'>
          <FieldContainer control={control} name='verbs' className='w-full'>
            {({ field }) => (
              <_Select
                label='VERBS'
                options={((resources || []).includes('stages')
                  ? availableVerbs.concat(['promote'])
                  : availableVerbs
                ).map((v) => ({ value: v, label: v }))}
                placeholder='create'
                value={field.value}
                onChange={field.onChange}
              />
            )}
          </FieldContainer>
          <FieldContainer control={control} name='resources' className='w-full'>
            {({ field }) => (
              <_Select
                label='RESOURCES'
                value={field.value}
                onChange={(value) => field.onChange(value)}
                placeholder='stages'
                className='w-full'
                options={availableResources.map((r) => ({ value: r, label: r }))}
              />
            )}
          </FieldContainer>
          <FieldContainer control={control} name='resourceNames' className='w-full'>
            {({ field }) => (
              <MultiStringEditor
                // @ts-expect-error zod infer problem
                value={field.value}
                onChange={field.onChange}
                placeholder='my-stage'
                label='Resource Names'
              />
            )}
          </FieldContainer>
        </div>

        <div className='flex items-end w-full'>
          <Button onClick={onSubmit} icon={<FontAwesomeIcon icon={faPlus} />} className='ml-auto'>
            Add Rule
          </Button>
        </div>
      </div>
    </div>
  );
};
