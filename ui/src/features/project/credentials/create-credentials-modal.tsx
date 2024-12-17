import { useMutation } from '@connectrpc/connect-query';
import { faAsterisk, faCode, faExternalLink } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import { Input, Modal, Segmented } from 'antd';
import { Controller, useForm } from 'react-hook-form';
import { z } from 'zod';

import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { SegmentLabel } from '@ui/features/common/segment-label';
import { dnsRegex } from '@ui/features/common/utils';
import { Secret } from '@ui/gen/k8s.io/api/core/v1/generated_pb';
import {
  createCredentials,
  updateCredentials
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { zodValidators } from '@ui/utils/validators';

import { SecretEditor } from './secret-editor';
import { CredentialsType } from './types';
import { constructDefaults, labelForKey, typeLabel } from './utils';

const createFormSchema = (editing?: boolean) =>
  z
    .object({
      name: zodValidators.requiredString.regex(
        dnsRegex,
        'Credentials name must be a valid DNS subdomain.'
      ),
      description: z.string().optional(),
      type: zodValidators.requiredString,
      repoUrl: zodValidators.requiredString,
      repoUrlIsRegex: z.boolean().optional(),
      username: zodValidators.requiredString,
      password: editing ? z.string().optional() : zodValidators.requiredString
    })
    .or(
      z.object({
        name: zodValidators.requiredString.regex(
          dnsRegex,
          'Credentials name must be a valid DNS subdomain.'
        ),
        description: z.string().optional(),
        type: zodValidators.requiredString,
        data: z.record(z.string(), z.string())
      })
    )
    .refine((data) => ['git', 'helm', 'image', 'generic'].includes(data.type), {
      message: "Type must be one of 'git', 'helm', 'image' or 'generic'."
    });

const placeholders = {
  name: 'My Credentials',
  description: 'An optional description',
  repoUrl: 'https://github.com/myusername/myrepo.git',
  username: 'admin',
  password: '********'
};

const genericCredentialPlaceholders = {
  name: 'My Secret',
  description: placeholders.description
};

const repoUrlPatternPlaceholder = '(?:https?://)?(?:www.)?github.com/[w.-]+/[w.-]+(?:.git)?';

type Props = ModalComponentProps & {
  project: string;
  onSuccess: () => void;
  init?: Secret;
  editing?: boolean;
  type: 'repo' | 'generic';
};

export const CreateCredentialsModal = ({ project, onSuccess, editing, init, ...props }: Props) => {
  const { control, handleSubmit, watch } = useForm({
    defaultValues: { ...constructDefaults(init, props.type === 'generic' ? props.type : 'git') },
    resolver: zodResolver(createFormSchema(editing))
  });

  const createCredentialsMutation = useMutation(createCredentials, {
    onSuccess: () => {
      props.hide();
      onSuccess();
    }
  });

  const updateCredentialsMutation = useMutation(updateCredentials, {
    onSuccess: () => {
      props.hide();
      onSuccess();
    }
  });

  const repoUrlIsRegex = watch('repoUrlIsRegex');

  const credentialType = (props.type === 'repo' ? watch('type') : 'generic') as CredentialsType;

  return (
    <Modal
      onCancel={props.hide}
      okButtonProps={{
        loading: createCredentialsMutation.isPending || updateCredentialsMutation.isPending
      }}
      okText={editing ? 'Update' : 'Create'}
      onOk={handleSubmit((values) => {
        if (editing) {
          return updateCredentialsMutation.mutate({
            ...values,
            project,
            name: init?.metadata?.name || ''
          });
        } else {
          createCredentialsMutation.mutate({ ...values, project });
        }
      })}
      title={
        <>
          <FontAwesomeIcon icon={faAsterisk} className='mr-2' />
          {editing ? 'Edit' : 'Create'} Secrets
        </>
      }
      {...props}
      width='612px'
    >
      {props.type === 'repo' && (
        <div className='mb-4'>
          <label className='block mb-2'>Type</label>
          <Controller
            name='type'
            control={control}
            render={({ field }) => (
              <Segmented
                className='w-full'
                block
                {...field}
                options={[
                  { label: typeLabel('git'), value: 'git' },
                  { label: typeLabel('helm'), value: 'helm' },
                  { label: typeLabel('image'), value: 'image' }
                ]}
                onChange={(newValue) => field.onChange(newValue)}
                value={field.value}
              />
            )}
          />
        </div>
      )}
      {Object.keys(credentialType === 'generic' ? genericCredentialPlaceholders : placeholders).map(
        (key) => (
          <div key={key}>
            {key === 'repoUrl' && (
              <>
                <label className='block mb-4'>Repo URL / Pattern</label>
                <Controller
                  name='repoUrlIsRegex'
                  control={control}
                  render={({ field }) => (
                    <Segmented
                      className='w-full mb-4'
                      block
                      {...field}
                      options={[
                        {
                          label: <SegmentLabel icon={faExternalLink}>URL</SegmentLabel>,
                          value: 'url'
                        },
                        {
                          label: <SegmentLabel icon={faCode}>Regex Pattern</SegmentLabel>,
                          value: 'regex'
                        }
                      ]}
                      onChange={(newValue) => field.onChange(newValue === 'regex')}
                      value={field.value ? 'regex' : 'url'}
                    />
                  )}
                />
              </>
            )}
            <FieldContainer
              label={key !== 'repoUrl' ? labelForKey(key) : undefined}
              name={key as 'name' | 'type' | 'repoUrl' | 'username' | 'password'}
              control={control}
            >
              {({ field }) => (
                // @ts-expect-error repoUrlInRegex won't be here so no boolean only strings
                <Input
                  {...field}
                  type={key === 'password' ? 'password' : 'text'}
                  placeholder={
                    key === 'repoUrl' && repoUrlIsRegex
                      ? repoUrlPatternPlaceholder
                      : placeholders[key as keyof typeof placeholders]
                  }
                  disabled={editing && key === 'name'}
                />
              )}
            </FieldContainer>
          </div>
        )
      )}
      {credentialType === 'generic' && (
        <FieldContainer control={control} name='data' label='Secrets'>
          {({ field }) => (
            <SecretEditor
              secret={field.value as Record<string, string>}
              onChange={field.onChange}
            />
          )}
        </FieldContainer>
      )}
    </Modal>
  );
};
