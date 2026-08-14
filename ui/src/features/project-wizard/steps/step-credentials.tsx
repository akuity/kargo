import {
  faCode,
  faExternalLink,
  faKey,
  faPlus,
  faQuestionCircle,
  faTrash
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import {
  Alert,
  Button,
  Card,
  Flex,
  Form,
  Input,
  Popover,
  Segmented,
  Space,
  Tag,
  Typography
} from 'antd';

import { SegmentLabel } from '@ui/features/common/segment-label';
import { iconForCredentialsType, typeLabel } from '@ui/features/common/settings/secrets/utils';
import { dnsRegex } from '@ui/features/common/utils';

import { CredentialAuthMethod, CredentialData, CredentialType, initialCredential } from '../types';

import { repoUrlError } from './credential-validation';

// Only shown when a credential's auth method was set by editing the YAML rail;
// the form itself (matching project settings) is username/password only.
const authLabels: Record<CredentialAuthMethod, string> = {
  userpass: 'Username + password / PAT',
  ssh: 'SSH private key (deprecated)',
  'github-app': 'GitHub App',
  'aws-ecr': 'AWS ECR access key',
  'gcp-ar': 'GCP Artifact Registry service account key',
  ambient: 'Ambient credentials'
};

// Mirrors the placeholders in the settings Create Credentials modal
const placeholders = {
  name: 'My Credentials',
  description: 'An optional description',
  username: 'admin',
  password: '********'
};

const repoUrlPlaceholders: Record<CredentialType, string> = {
  git: 'https://github.com/myusername/myrepo.git',
  helm: 'ghcr.io/nginxinc/charts/nginx-ingress',
  image: 'public.ecr.aws/nginx/nginx'
};

const repoUrlPatternPlaceholder = '(?:https?://)?(?:www.)?github.com/[w.-]+/[w.-]+(?:.git)?';

type CredentialCardProps = {
  cred: CredentialData;
  onChange: (cred: CredentialData) => void;
  onRemove: () => void;
};

const RepoCredentialCard = ({ cred, onChange, onRemove }: CredentialCardProps) => {
  const patch = (p: Partial<CredentialData>) => onChange({ ...cred, ...p });

  const nameError =
    cred.name && !dnsRegex.test(cred.name)
      ? 'Credentials name must be a valid DNS subdomain.'
      : undefined;

  return (
    <Card
      title={
        <span className='flex items-center gap-2'>
          <FontAwesomeIcon icon={iconForCredentialsType(cred.type)} className='text-gray-400' />
          <span className='font-mono text-sm'>{cred.name || '(unnamed)'}</span>
          <Tag className='mr-0'>{cred.type}</Tag>
        </span>
      }
      extra={
        <Button
          type='text'
          danger
          size='small'
          icon={<FontAwesomeIcon icon={faTrash} size='sm' />}
          onClick={onRemove}
        />
      }
    >
      <div className='mb-4'>
        <label className='block mb-2'>Type</label>
        <Segmented
          size='small'
          value={cred.type}
          options={(['git', 'helm', 'image'] as CredentialType[]).map((type) => ({
            label: typeLabel(type),
            value: type
          }))}
          onChange={(type) => patch({ type: type as CredentialType })}
        />
      </div>
      <Form layout='vertical' component='div'>
        <Form.Item label='Name' validateStatus={nameError ? 'error' : ''} help={nameError}>
          <Input
            placeholder={placeholders.name}
            value={cred.name}
            onChange={(e) => patch({ name: e.target.value })}
          />
        </Form.Item>
        <Form.Item label='Description'>
          <Input
            placeholder={placeholders.description}
            value={cred.description}
            onChange={(e) => patch({ description: e.target.value })}
          />
        </Form.Item>
        <label className='block mb-4'>Repo URL / Pattern</label>
        <Segmented
          size='small'
          className='mb-4'
          value={cred.repoURLIsRegex ? 'regex' : 'url'}
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
          onChange={(value) => patch({ repoURLIsRegex: value === 'regex' })}
        />
        <Form.Item validateStatus={repoUrlError(cred) ? 'error' : ''} help={repoUrlError(cred)}>
          <Input
            placeholder={
              cred.repoURLIsRegex ? repoUrlPatternPlaceholder : repoUrlPlaceholders[cred.type]
            }
            value={cred.repoURL}
            onChange={(e) => patch({ repoURL: e.target.value })}
          />
        </Form.Item>
        {cred.auth === 'userpass' ? (
          <>
            <Form.Item label='Username'>
              <Input
                placeholder={placeholders.username}
                value={cred.username}
                onChange={(e) => patch({ username: e.target.value })}
              />
            </Form.Item>
            <Form.Item label='Password'>
              <Input
                type='password'
                placeholder={placeholders.password}
                value={cred.password}
                onChange={(e) => patch({ password: e.target.value })}
              />
            </Form.Item>
          </>
        ) : (
          <Alert
            type='info'
            showIcon
            message={
              <>
                This credential uses <strong>{authLabels[cred.auth]}</strong> authentication,
                configured through the YAML editor. Edit it there.
              </>
            }
          />
        )}
      </Form>
    </Card>
  );
};

type StepCredentialsProps = {
  value: CredentialData[];
  onChange: (value: CredentialData[]) => void;
};

export const StepCredentials = ({ value, onChange }: StepCredentialsProps) => {
  const add = () => onChange([...value, initialCredential('git')]);

  return (
    <Card
      type='inner'
      title={
        <Space size={4}>
          Repository Credentials
          <Popover content='These credentials are used implicitly by Warehouses and promotion steps that match the repository URL or pattern.'>
            <Typography.Text type='secondary'>
              <FontAwesomeIcon icon={faQuestionCircle} size='xs' />
            </Typography.Text>
          </Popover>
        </Space>
      }
      extra={
        <Button icon={<FontAwesomeIcon icon={faPlus} />} onClick={add}>
          Add Credentials
        </Button>
      }
    >
      {value.length === 0 ? (
        <div className='flex flex-col items-center py-10 text-center'>
          <FontAwesomeIcon icon={faKey} className='text-2xl text-gray-300 mb-3' />
          <div className='text-sm text-gray-500 max-w-md'>
            Add credentials if your Warehouses will pull from a private repository the cluster
            cannot reach with ambient credentials. Otherwise, continue — you can add them later from
            project settings.
          </div>
        </div>
      ) : (
        <Flex gap={16} vertical>
          {value.map((cred, i) => (
            <RepoCredentialCard
              key={i}
              cred={cred}
              onChange={(next) => onChange(value.map((c, x) => (x === i ? next : c)))}
              onRemove={() => onChange(value.filter((_, x) => x !== i))}
            />
          ))}
        </Flex>
      )}
    </Card>
  );
};
