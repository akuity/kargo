import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Drawer, Input } from 'antd';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { FieldContainer } from '@ui/features/common/form/field-container';
import { MultiStringEditor } from '@ui/features/common/form/multi-string-editor';
import { DESCRIPTION_ANNOTATION_KEY, dnsRegex } from '@ui/features/common/utils';
import { Claim, RbacRole, V1PolicyRule } from '@ui/gen/api/v2/models';
import { useCreateProjectRole, useUpdateRole } from '@ui/gen/api/v2/rbac/rbac';
import { zodValidators } from '@ui/utils/validators';

import { RuleEditor } from './rule-editor';
import { RulesTable } from './rules-table';

type Props = {
  project: string;
  onSuccess: () => void;
  editing?: RbacRole;
  hide: () => void;
};

type AllowedFields = 'name' | 'email' | 'sub' | 'groups';

const annotationsWithDescription = (description: string): { [key: string]: string } => {
  return description ? { [DESCRIPTION_ANNOTATION_KEY]: description } : {};
};

const nonZeroArray = (name: string) =>
  z.array(z.string()).min(0, `At least one ${name} is required`);

const formSchema = z.object({
  name: zodValidators.requiredString.regex(dnsRegex, 'Role name must be a valid DNS subdomain.'),
  description: z.string().optional(),
  email: nonZeroArray('email'),
  sub: nonZeroArray('sub'),
  groups: nonZeroArray('groups')
});

const multiFields: { name: AllowedFields; label?: string; placeholder: string }[] = [
  { name: 'email', placeholder: 'email@corp.com' },
  { name: 'sub', label: 'Subjects', placeholder: 'mysubject' },
  { name: 'groups', placeholder: 'mygroup' }
];

export const CreateRole = ({ editing, onSuccess, project, hide }: Props) => {
  const { control, handleSubmit } = useForm({
    resolver: zodResolver(formSchema),
    values: {
      name: editing?.metadata?.name || '',
      description: editing?.metadata?.annotations?.[DESCRIPTION_ANNOTATION_KEY] || '',
      email: editing?.claims?.find((claim: Claim) => claim.name === 'email')?.values || [],
      sub: editing?.claims?.find((claim: Claim) => claim.name === 'sub')?.values || [],
      groups: editing?.claims?.find((claim: Claim) => claim.name === 'groups')?.values || []
    }
  });

  const { mutate } = useCreateProjectRole({
    mutation: {
      onSuccess: () => {
        hide();
        onSuccess();
      }
    }
  });

  const { mutate: update } = useUpdateRole({
    mutation: {
      onSuccess: () => {
        hide();
        onSuccess();
      }
    }
  });

  const onSubmit = handleSubmit((values) => {
    const annotations = annotationsWithDescription(values.description || '');
    const getClaims = (): Claim[] => {
      const claimsArray: Claim[] = [];
      multiFields.map((field) => {
        const newClaim: Claim = { name: String(field.name) };
        if (newClaim.name === 'email') {
          if (values.email.length === 0) {
            return;
          }
          newClaim.values = values.email;
        } else if (newClaim.name === 'sub') {
          if (values.sub.length === 0) {
            return;
          }
          newClaim.values = values.sub;
        } else if (newClaim.name === 'groups') {
          if (values.groups.length === 0) {
            return;
          }
          newClaim.values = values.groups;
        } else {
          if (values[field.name].length === 0) {
            return;
          }
          newClaim.values = values[field.name] as string[];
        }
        claimsArray.push(newClaim);
      });
      return claimsArray;
    };
    if (editing) {
      return update({
        project,
        role: editing?.metadata?.name || '',
        data: {
          ...values,
          rules,
          metadata: { namespace: project, name: editing?.metadata?.name, annotations },
          claims: getClaims()
        } as unknown as { [key: string]: unknown }
      });
    } else {
      mutate({
        project,
        data: {
          ...values,
          rules,
          metadata: { name: values.name, namespace: project, annotations },
          claims: getClaims()
        } as unknown as { [key: string]: unknown }
      });
    }
  });

  const [rules, setRules] = useState<V1PolicyRule[]>(editing?.rules || []);

  return (
    <Drawer open={true} onClose={hide} width='80%' title={`${editing ? 'Edit' : 'Create'} Role`}>
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
        <FieldContainer
          label='Description'
          name='description'
          control={control}
          formItemOptions={{ className: 'mb-4' }}
        >
          {({ field }) => (
            <Input {...field} type='text' placeholder='An optional description of this role' />
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
      <Button type='primary' onClick={onSubmit}>
        Save
      </Button>
    </Drawer>
  );
};
