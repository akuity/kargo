import { useMutation } from '@connectrpc/connect-query';
import { faIdBadge } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import { Input, Modal, Segmented } from 'antd';
import { Controller, useForm } from 'react-hook-form';
import { z } from 'zod';

import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import {
  createCredentials,
  updateCredentials
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Secret } from '@ui/gen/v1alpha1/types_pb';
import { zodValidators } from '@ui/utils/validators';

import {
  CredentialTypeLabelKey,
  CredentialsType,
  IconForCredentialsType
} from './credentials-list';

const credentialsNameRegex = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$/;

const formSchema = z
  .object({
    name: zodValidators.requiredString.regex(
      credentialsNameRegex,
      'Credentials name must be a valid DNS subdomain.'
    ),
    type: zodValidators.requiredString,
    repoUrl: z.string().optional(),
    repoUrlPattern: z.string().optional(),
    username: zodValidators.requiredString,
    password: zodValidators.requiredString
  })
  .refine((data) => data.repoUrl || data.repoUrlPattern, {
    message: "Either 'repoUrl' or 'repoUrlPattern' must be set."
  })
  .refine((data) => ['git', 'helm', 'image'].includes(data.type), {
    message: "Type must be one of 'git', 'helm', or 'image'."
  });

const placeholders = {
  name: 'My Credentials',
  repoUrl: 'https://github.com/myusername/myrepo.git',
  repoUrlPattern: '(?:https?://)?(?:www.)?github.com/[w.-]+/[w.-]+(?:.git)?',
  username: 'admin',
  password: 'admin12345'
};

const typeLabel = (type: CredentialsType) => (
  <span className='flex items-center font-semibold justify-center text-center p-4'>
    <FontAwesomeIcon icon={IconForCredentialsType(type)} className='mr-2' />
    {type.toUpperCase()}
  </span>
);

const labelForKey = (s: string) =>
  s
    .split('')
    .map((c, i) => (c === c.toUpperCase() ? (i !== 0 ? ' ' : '') + c : c))
    .join('')
    .replace(/^./, (str) => str.toUpperCase())
    .replace('Url', 'URL');

const constructDefaults = (init?: Secret) => {
  if (!init) {
    return {
      name: '',
      type: 'git',
      repoUrl: '',
      repoUrlPattern: '',
      username: '',
      password: ''
    };
  }
  return {
    name: init?.metadata?.name || '',
    type: init?.metadata?.labels[CredentialTypeLabelKey] || 'git',
    repoUrl: init?.stringData['repoURL'],
    repoUrlPattern: init?.stringData['repoURLPattern'],
    username: init?.stringData['username'],
    password: ''
  };
};

type Props = ModalComponentProps & {
  project: string;
  onSuccess: () => void;
  init?: Secret;
  editing?: boolean;
};

export const CreateCredentialsModal = ({ project, onSuccess, editing, init, ...props }: Props) => {
  const { control, handleSubmit } = useForm({
    defaultValues: constructDefaults(init),
    resolver: zodResolver(formSchema)
  });

  const { mutate } = useMutation(createCredentials, {
    onSuccess: () => {
      props.hide();
      onSuccess();
    }
  });

  const { mutate: update } = useMutation(updateCredentials, {
    onSuccess: () => {
      props.hide();
      onSuccess();
    }
  });

  return (
    <Modal
      onCancel={props.hide}
      onOk={handleSubmit((values) => {
        if (editing) {
          return update({ ...values, project, name: init?.metadata?.name || '' });
        } else {
          mutate({ ...values, project });
        }
      })}
      title={
        <>
          <FontAwesomeIcon icon={faIdBadge} className='mr-2' />
          {editing ? 'Edit' : 'Create'} Credentials
        </>
      }
      {...props}
    >
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
      {Object.keys(placeholders).map((key) => (
        <FieldContainer
          key={key}
          label={labelForKey(key)}
          name={key as 'name' | 'type' | 'repoUrl' | 'repoUrlPattern' | 'username' | 'password'}
          control={control}
        >
          {({ field }) => (
            <Input
              {...field}
              type={key === 'password' ? 'password' : 'text'}
              placeholder={placeholders[key as keyof typeof placeholders]}
              disabled={editing && key === 'name'}
            />
          )}
        </FieldContainer>
      ))}
    </Modal>
  );
};
