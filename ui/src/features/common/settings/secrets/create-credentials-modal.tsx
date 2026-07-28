import { faAsterisk, faCode, faExternalLink } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { AutoComplete, Checkbox, Input, Modal, Segmented, Typography } from 'antd';
import { Controller, useForm } from 'react-hook-form';

import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { SegmentLabel } from '@ui/features/common/segment-label';
import {
  createProjectGenericCredentials,
  createProjectRepoCredentials,
  createSharedGenericCredentials,
  createSharedRepoCredentials,
  updateProjectGenericCredentials,
  updateProjectRepoCredentials,
  updateSharedGenericCredentials,
  updateSharedRepoCredentials
} from '@ui/gen/api/v2/credentials/credentials';
import { V1Secret } from '@ui/gen/api/v2/models';

import { createFormSchema, SecretFormValues } from './schema-validator';
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

// Common Kubernetes Secret types offered as suggestions. The field is free-form,
// so any valid Secret type may be entered. Secret type is immutable and can only
// be set at creation time.
const k8sSecretTypeOptions = [
  'Opaque',
  'kubernetes.io/dockerconfigjson',
  'kubernetes.io/dockercfg',
  'kubernetes.io/basic-auth',
  'kubernetes.io/ssh-auth',
  'kubernetes.io/tls',
  'kubernetes.io/service-account-token'
].map((value) => ({ value }));

const repoUrlPatternPlaceholder = '(?:https?://)?(?:www.)?github.com/[w.-]+/[w.-]+(?:.git)?';

type Props = ModalComponentProps & {
  project: string;
  onSuccess: () => void;
  init?: V1Secret;
  editing?: boolean;
  type: 'repo' | 'generic';
};

export const CreateCredentialsModal = ({ project, onSuccess, editing, init, ...props }: Props) => {
  const { control, handleSubmit, watch } = useForm<SecretFormValues>({
    defaultValues: constructDefaults(init, props.type === 'generic' ? props.type : 'git'),
    resolver: zodResolver(createFormSchema(props.type === 'generic', editing))
  });

  const onMutationSuccess = () => {
    props.hide();
    onSuccess();
  };

  const createRepoMutation = useMutation({
    mutationFn: (values: ReturnType<typeof constructDefaults>) => {
      const body = {
        name: values.name,
        type: values.type,
        repoUrl: values.repoUrl,
        repoUrlIsRegex: values.repoUrlIsRegex,
        username: values.username,
        password: values.password,
        description: values.description
      };
      return project
        ? createProjectRepoCredentials(project, body)
        : createSharedRepoCredentials(body);
    },
    onSuccess: onMutationSuccess
  });

  const updateRepoMutation = useMutation({
    mutationFn: (values: ReturnType<typeof constructDefaults>) => {
      const name = init?.metadata?.name || '';
      const body = {
        type: values.type,
        repoUrl: values.repoUrl,
        repoUrlIsRegex: values.repoUrlIsRegex,
        username: values.username,
        password: values.password,
        description: values.description
      };
      return project
        ? updateProjectRepoCredentials(project, name, body)
        : updateSharedRepoCredentials(name, body);
    },
    onSuccess: onMutationSuccess
  });

  const createGenericMutation = useMutation({
    mutationFn: (payload: {
      name: string;
      data: Record<string, string>;
      description?: string;
      replicate?: boolean;
      type?: string;
    }) =>
      project
        ? createProjectGenericCredentials(project, payload)
        : createSharedGenericCredentials(payload),
    onSuccess: onMutationSuccess
  });

  const updateGenericMutation = useMutation({
    mutationFn: (payload: {
      data: Record<string, string>;
      description?: string;
      replicate?: boolean;
    }) => {
      const name = init?.metadata?.name || '';
      return project
        ? updateProjectGenericCredentials(project, name, payload)
        : updateSharedGenericCredentials(name, payload);
    },
    onSuccess: onMutationSuccess
  });

  const repoUrlIsRegex = watch('repoUrlIsRegex');

  const credentialType = (props.type === 'repo' ? watch('type') : 'generic') as CredentialsType;

  const onSubmit = handleSubmit((values) => {
    if (credentialType === 'generic') {
      const data: Record<string, string> = {};

      for (const [k, v] of values.data ?? []) {
        data[k] = v;
      }

      const replicate = values.replicate;

      if (editing) {
        return updateGenericMutation.mutate({ data, description: values.description, replicate });
      }

      return createGenericMutation.mutate({
        name: values.name,
        data,
        description: values.description,
        replicate,
        type: values.secretType
      });
    }

    if (editing) {
      return updateRepoMutation.mutate(values);
    }

    return createRepoMutation.mutate(values);
  });

  const isPending =
    createRepoMutation.isPending ||
    updateRepoMutation.isPending ||
    createGenericMutation.isPending ||
    updateGenericMutation.isPending;

  return (
    <Modal
      okButtonProps={{ loading: isPending }}
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
        <div className='mb-4'>
          <label className='block mb-2'>Type</label>
          <Controller
            name='secretType'
            control={control}
            render={({ field }) => (
              <AutoComplete
                className='w-full'
                options={k8sSecretTypeOptions}
                placeholder='Opaque'
                disabled={editing}
                filterOption={(inputValue, option) =>
                  (option?.value ?? '').toLowerCase().includes(inputValue.toLowerCase())
                }
                value={field.value}
                onChange={(value) => field.onChange(value)}
              />
            )}
          />
          <Typography.Text type='secondary' className='text-xs'>
            The Kubernetes Secret type
          </Typography.Text>
        </div>
      )}
      {credentialType === 'generic' && !project && (
        <Controller
          name='replicate'
          control={control}
          render={({ field }) => (
            <label className='flex items-start gap-2 cursor-pointer mb-4'>
              <Checkbox
                checked={!!field.value}
                onChange={(e) => field.onChange(e.target.checked)}
              />
              <div>
                <div>Replicate</div>
                <Typography.Text type='secondary'>
                  Replicate the resource to all projects to be used by AnalysisTemplates
                </Typography.Text>
              </div>
            </label>
          )}
        />
      )}
      {credentialType === 'generic' && (
        <FieldContainer control={control} name='data' label='Data'>
          {({ field }) => <SecretEditor secret={field.value ?? []} onChange={field.onChange} />}
        </FieldContainer>
      )}
    </Modal>
  );
};
