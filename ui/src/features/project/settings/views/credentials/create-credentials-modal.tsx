import { useMutation } from '@connectrpc/connect-query';
import { faAsterisk, faCode, faExternalLink } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import { Input, Modal, Segmented } from 'antd';
import { Controller, useForm } from 'react-hook-form';

import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { SegmentLabel } from '@ui/features/common/segment-label';
import {
  createRepoCredentials,
  createGenericCredentials,
  updateRepoCredentials,
  updateGenericCredentials
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Secret } from '@ui/gen/k8s.io/api/core/v1/generated_pb';

import { createFormSchema } from './schema-validator';
import { SecretEditor } from './secret-editor';
import { CredentialsType } from './types';
import { constructDefaults, labelForKey, typeLabel } from './utils';

const placeholders = {
  name: 'My Credentials',
  description: 'An optional description',
  repoUrl: 'https://github.com/myusername/myrepo.git',
  username: 'admin',
  password: '********'
};

const repoUrlPlaceholder = (credType: CredentialsType) => {
  switch (credType) {
    case 'git':
      return placeholders.repoUrl;
    case 'helm':
      return 'ghcr.io/nginxinc/charts/nginx-ingress';
    case 'image':
      return 'public.ecr.aws/nginx/nginx';
  }

  return '';
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
    resolver: zodResolver(createFormSchema(props.type === 'generic', editing))
  });

  const createCredentialsMutation = useMutation(createRepoCredentials, {
    onSuccess: () => {
      props.hide();
      onSuccess();
    }
  });

  const updateCredentialsMutation = useMutation(updateRepoCredentials, {
    onSuccess: () => {
      props.hide();
      onSuccess();
    }
  });

  const createSecretsMutation = useMutation(createGenericCredentials, {
    onSuccess: () => {
      props.hide();
      onSuccess();
    }
  });

  const updateSecretsMutation = useMutation(updateGenericCredentials, {
    onSuccess: () => {
      props.hide();
      onSuccess();
    }
  });

  const repoUrlIsRegex = watch('repoUrlIsRegex');

  const credentialType = (props.type === 'repo' ? watch('type') : 'generic') as CredentialsType;

  const onSubmit = handleSubmit((values) => {
    if (credentialType === 'generic') {
      const data: Record<string, string> = {};

      // @ts-expect-error zod infer problem
      if (values?.data?.length > 0) {
        // @ts-expect-error zod infer problem
        for (const [k, v] of values.data) {
          data[k] = v;
        }
      }

      if (editing) {
        return updateSecretsMutation.mutate({
          ...values,
          project,
          name: init?.metadata?.name || '',
          data
        });
      }

      return createSecretsMutation.mutate({ ...values, project, data });
    }

    if (editing) {
      return updateCredentialsMutation.mutate({
        ...values,
        project,
        name: init?.metadata?.name || ''
      });
    }

    return createCredentialsMutation.mutate({ ...values, project });
  });

  return (
    <Modal
      okButtonProps={{
        loading: createCredentialsMutation.isPending || updateCredentialsMutation.isPending
      }}
      okText={editing ? 'Update' : 'Create'}
      onOk={onSubmit}
      title={
        <>
          <FontAwesomeIcon icon={faAsterisk} className='mr-2' />
          {editing ? 'Edit' : 'Create'} {props.type === 'repo' ? 'Credentials' : 'Secret'}
        </>
      }
      onCancel={props.hide}
      open={props.visible}
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
                <Input
                  {...field}
                  type={key === 'password' ? 'password' : 'text'}
                  placeholder={
                    key === 'repoUrl' && repoUrlIsRegex
                      ? repoUrlPatternPlaceholder
                      : key === 'repoUrl'
                        ? repoUrlPlaceholder(credentialType)
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
        // @ts-expect-error expected type is there
        <FieldContainer control={control} name='data' label='Data'>
          {({ field }) => (
            // @ts-expect-error expectedtype is there
            <SecretEditor secret={field.value as [string, string][]} onChange={field.onChange} />
          )}
        </FieldContainer>
      )}
    </Modal>
  );
};
