import { faAnchor, faBoxesStacked, faFileCode } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import { Col, Flex, Input, Radio, Row, Switch, Typography } from 'antd';
import { useState } from 'react';
import { useForm } from 'react-hook-form';

import { FieldContainer } from '@ui/features/common/form/field-container';

import { HelmMechanism } from './helm-mechanism';
import { KustomizeMechanism } from './kustomize-mechanism';
import { RenderMechanism } from './render-mechanism';
import { gitRepoUpdateSchema } from './schemas';

export const GitUpdateEditor = () => {
  const { control, handleSubmit } = useForm({
    resolver: zodResolver(gitRepoUpdateSchema)
  });

  const [selectedMechanism, setSelectedMechanism] = useState<'render' | 'kustomize' | 'helm'>(
    'render'
  );

  return (
    <div>
      <Typography.Title level={4}>Git Repo Update</Typography.Title>
      <FieldContainer label='Repo URL' name='repoUrl' control={control} required>
        {({ field }) => <Input {...field} placeholder='https://github.com/my-org/my-repo' />}
      </FieldContainer>
      <FieldContainer name='insecureSkipVerify' control={control}>
        {({ field }) => (
          <Flex align='center'>
            <Switch checked={field.value as boolean} onChange={(value) => field.onChange(value)} />
            <div className='ml-2'>Insecure Skip Verify</div>
          </Flex>
        )}
      </FieldContainer>
      <Row gutter={16}>
        <Col span={12}>
          <FieldContainer label='Read Branch' name='readBranch' control={control}>
            {({ field }) => <Input {...field} placeholder='main' />}
          </FieldContainer>
        </Col>
        <Col span={12}>
          <FieldContainer label='Write Branch' name='writeBranch' control={control}>
            {({ field }) => <Input {...field} placeholder='main' />}
          </FieldContainer>
        </Col>
      </Row>
      <FieldContainer
        name='pullRequest.type'
        label='Pull Request'
        control={control}
        description='If enabled, will generate a Pull Request instead of making a commit directly.'
      >
        {({ field }) => (
          <Radio.Group {...field}>
            <Radio.Button>Disabled</Radio.Button>
            <Radio.Button value='github'>GitHub</Radio.Button>
            <Radio.Button value='gitlab'>GitLab</Radio.Button>
          </Radio.Group>
        )}
      </FieldContainer>

      <Typography.Title level={4} className='!mb-4'>
        Mechanism
      </Typography.Title>
      <Radio.Group
        value={selectedMechanism}
        onChange={(e) => setSelectedMechanism(e.target.value)}
        className='mb-4'
      >
        <Radio.Button value='render'>
          <FontAwesomeIcon icon={faBoxesStacked} className='mr-2' />
          Kargo Render
        </Radio.Button>
        <Radio.Button value='kustomize'>
          <FontAwesomeIcon icon={faFileCode} className='mr-2' />
          Kustomize
        </Radio.Button>
        <Radio.Button value='helm'>
          <FontAwesomeIcon icon={faAnchor} className='mr-2' />
          Helm
        </Radio.Button>
      </Radio.Group>

      {selectedMechanism === 'render' && <RenderMechanism />}
      {selectedMechanism === 'kustomize' && <KustomizeMechanism />}
      {selectedMechanism === 'helm' && <HelmMechanism />}
    </div>
  );
};
