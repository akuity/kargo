import { useMutation } from '@connectrpc/connect-query';
import { faPeopleGroup } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Drawer, Input, Typography } from 'antd';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { FieldContainer } from '@ui/features/common/form/field-container';
import { MultiStringEditor } from '@ui/features/common/form/multi-string-editor';
import { dnsRegex } from '@ui/features/common/utils';
import { PolicyRule } from '@ui/gen/k8s.io/api/rbac/v1/generated_pb';
import { Role } from '@ui/gen/rbac/v1alpha1/generated_pb';
import { createRole, updateRole } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { zodValidators } from '@ui/utils/validators';

import { RuleEditor } from './rule-editor';
import { RulesTable } from './rules-table';

type Props = {
  project: string;
  onSuccess: () => void;
  editing?: Role;
  hide: () => void;
};

type AllowedFields = 'name' | 'emails' | 'subs' | 'groups';

const nonZeroArray = (name: string) =>
  z.array(z.string()).min(0, `At least one ${name} is required`);

const formSchema = z.object({
  name: zodValidators.requiredString.regex(dnsRegex, 'Role name must be a valid DNS subdomain.'),
  emails: nonZeroArray('email'),
  subs: nonZeroArray('sub'),
  groups: nonZeroArray('group')
});

const multiFields: { name: AllowedFields; label?: string; placeholder: string }[] = [
  { name: 'emails', placeholder: 'email@corp.com' },
  { name: 'subs', label: 'Subjects', placeholder: 'mysubject' },
  { name: 'groups', placeholder: 'mygroup' }
];

export const CreateRole = ({ editing, onSuccess, project, hide }: Props) => {
  const { control, handleSubmit } = useForm({
    resolver: zodResolver(formSchema),
    values: {
      name: editing?.metadata?.name || '',
      emails: editing?.emails || [],
      subs: editing?.subs || [],
      groups: editing?.groups || []
    }
  });

  const { mutate } = useMutation(createRole, {
    onSuccess: () => {
      hide();
      onSuccess();
    }
  });

  const { mutate: update } = useMutation(updateRole, {
    onSuccess: () => {
      hide();
      onSuccess();
    }
  });

  const onSubmit = handleSubmit((values) => {
    if (editing) {
      return update({
        role: { ...values, rules, metadata: { namespace: project, name: editing?.metadata?.name } }
      });
    } else {
      mutate({ role: { ...values, rules, metadata: { name: values.name, namespace: project } } });
    }
  });

  const [rules, setRules] = useState<PolicyRule[]>(editing?.rules || []);

  return (
    <Drawer open={true} onClose={() => hide()} width={'85%'} closable={false}>
      <Typography.Title
        level={2}
        style={{ margin: 0, marginBottom: '0.5em' }}
        className='flex items-center'
      >
        <FontAwesomeIcon icon={faPeopleGroup} className='mr-2' />
        {editing ? 'Edit' : 'Create'} Role
        <Button type='primary' className='ml-auto' onClick={onSubmit}>
          Save
        </Button>
      </Typography.Title>
      <div className='mb-6'>
        <FieldContainer
          label='Name'
          name='name'
          control={control}
          formItemOptions={{ className: 'mb-4' }}
        >
          {({ field }) => (
            <Input {...field} type='text' placeholder='my-role' disabled={!!editing} />
          )}
        </FieldContainer>
        <div className='text-lg font-semibold mb-4'>OIDC Bindings</div>
        <div className='flex items-start gap-4'>
          {multiFields.map(({ name, placeholder, label }) => (
            <FieldContainer
              name={name}
              control={control}
              key={name}
              className='w-1/3'
              formItemOptions={{ className: 'mb-3' }}
            >
              {({ field }) => (
                <MultiStringEditor
                  value={field.value as string[]}
                  onChange={field.onChange}
                  placeholder={placeholder}
                  label={label ? label : name}
                />
              )}
            </FieldContainer>
          ))}
        </div>
      </div>
      <div>
        <div className='text-lg font-semibold mb-4'>Rules</div>
        <div className='flex gap-4'>
          <RulesTable rules={rules} setRules={setRules} />
          <RuleEditor onSuccess={(rule) => setRules([...rules, rule])} style={{ width: '600px' }} />
        </div>
      </div>
    </Drawer>
  );
};
