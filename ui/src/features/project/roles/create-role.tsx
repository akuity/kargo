import { create } from '@bufbuild/protobuf';
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
import { DESCRIPTION_ANNOTATION_KEY, dnsRegex } from '@ui/features/common/utils';
import { Claim, ClaimSchema, Role } from '@ui/gen/api/rbac/v1alpha1/generated_pb';
import {
  createRole,
  updateRole
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { PolicyRule } from '@ui/gen/k8s.io/api/rbac/v1/generated_pb';
import { zodValidators } from '@ui/utils/validators';

import { RuleEditor } from './rule-editor';
import { RulesTable } from './rules-table';

type Props = {
  project: string;
  onSuccess: () => void;
  editing?: Role;
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
      description: editing?.metadata?.annotations[DESCRIPTION_ANNOTATION_KEY] || '',
      email:
        editing?.claims.find((claim: Claim) => {
          if (claim.name === 'email') {
            return claim;
          } else {
            return undefined;
          }
        })?.values || [],
      sub:
        editing?.claims.find((claim: Claim) => {
          if (claim.name === 'sub') {
            return claim;
          } else {
            return undefined;
          }
        })?.values || [],
      groups:
        editing?.claims.find((claim: Claim) => {
          if (claim.name === 'groups') {
            return claim;
          } else {
            return undefined;
          }
        })?.values || []
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
    const annotations = annotationsWithDescription(values.description);
    const getClaims = (): Claim[] => {
      const claimsArray: Claim[] = [];
      multiFields.map((field) => {
        const newClaim = create(ClaimSchema, {
          name: String(field.name)
        });
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
        role: {
          ...values,
          rules,
          metadata: { namespace: project, name: editing?.metadata?.name, annotations },
          claims: getClaims()
        }
      });
    } else {
      mutate({
        role: {
          ...values,
          rules,
          metadata: { name: values.name, namespace: project, annotations },
          claims: getClaims()
        }
      });
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
    </Drawer>
  );
};
