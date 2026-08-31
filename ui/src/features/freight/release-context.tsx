import { faCodeCommit, faExternalLink } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import {
  Alert,
  Button,
  Card,
  Col,
  Collapse,
  Descriptions,
  Empty,
  Flex,
  Row,
  Skeleton,
  Space,
  Tag,
  Typography
} from 'antd';
import { formatDistance } from 'date-fns';
import { z } from 'zod';

import { useGetFreightReleaseContextConfig } from '@ui/gen/api/v2/core/core';
import { Freight, FreightReference } from '@ui/gen/api/v2/models';

import {
  getFreightImageReleaseContexts,
  ImageReleaseContext,
  pairImageReleaseContexts,
  PairedImageReleaseContext
} from './release-context-utils';

type ReleaseContextProps = {
  freight: Freight;
  currentFreight?: Freight | FreightReference;
  comparison?: boolean;
};

const timestampSchema = z.iso.datetime({ offset: true });

const formatTimestamp = (value: string): string => {
  if (!timestampSchema.safeParse(value).success) {
    return value;
  }
  const date = new Date(value);
  return `${formatDistance(date, new Date(), { addSuffix: true })} (${date.toLocaleString()})`;
};

const CopyableValue = ({ children }: { children?: string }) =>
  children ? <Typography.Text copyable>{children}</Typography.Text> : null;

const ImageDetails = ({ context }: { context?: ImageReleaseContext }) => {
  if (!context) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No image' />;
  }

  const { image } = context;
  const items = [
    ...(image.tag
      ? [{ key: 'tag', label: 'Tag', children: <CopyableValue>{image.tag}</CopyableValue> }]
      : []),
    ...(image.digest
      ? [
          {
            key: 'digest',
            label: 'Digest',
            children: <CopyableValue>{image.digest}</CopyableValue>
          }
        ]
      : []),
    ...(context.subject
      ? [
          {
            key: 'subject',
            label: 'Commit',
            children: context.subject
          }
        ]
      : []),
    ...(context.source
      ? [
          {
            key: 'source',
            label: 'Source',
            children: context.sourceURL ? (
              <Typography.Link href={context.sourceURL} target='_blank' rel='noopener noreferrer'>
                {context.source}
              </Typography.Link>
            ) : (
              <CopyableValue>{context.source}</CopyableValue>
            )
          }
        ]
      : []),
    ...(context.revision
      ? [
          {
            key: 'revision',
            label: 'Revision',
            children: <CopyableValue>{context.revision}</CopyableValue>
          }
        ]
      : []),
    ...(context.author ? [{ key: 'author', label: 'Author', children: context.author }] : []),
    ...(context.committer
      ? [{ key: 'committer', label: 'Committer', children: context.committer }]
      : []),
    ...(context.createdAt
      ? [
          {
            key: 'built',
            label: 'Built',
            children: formatTimestamp(context.createdAt)
          }
        ]
      : []),
    ...(context.commitCreatedAt
      ? [
          {
            key: 'commit-created',
            label: 'Committed',
            children: formatTimestamp(context.commitCreatedAt)
          }
        ]
      : [])
  ];

  return (
    <Space direction='vertical' size='middle' className='w-full'>
      <Descriptions column={1} bordered size='small' items={items} />
      {(context.sourceURL || context.commitURL) && (
        <Flex justify='end' gap='small' wrap>
          {context.sourceURL && (
            <Button
              href={context.sourceURL}
              target='_blank'
              rel='noopener noreferrer'
              icon={<FontAwesomeIcon icon={faExternalLink} />}
            >
              View source
            </Button>
          )}
          {context.commitURL && context.commitURL !== context.sourceURL && (
            <Button
              href={context.commitURL}
              target='_blank'
              rel='noopener noreferrer'
              icon={<FontAwesomeIcon icon={faCodeCommit} />}
            >
              View commit
            </Button>
          )}
        </Flex>
      )}
      {context.annotations.length > 0 && (
        <Collapse
          size='small'
          items={[
            {
              key: 'annotations',
              label: 'Image annotations',
              children: (
                <Flex gap='small' wrap>
                  {context.annotations.map(({ key, value }) => (
                    <Tag key={key} className='whitespace-normal break-all'>
                      {key}: {value}
                    </Tag>
                  ))}
                </Flex>
              )
            }
          ]}
        />
      )}
    </Space>
  );
};

const imageName = (pair: PairedImageReleaseContext): string =>
  pair.incoming?.image.repoURL || pair.current?.image.repoURL || 'Container image';

const statusColor = (status: PairedImageReleaseContext['status']): string | undefined => {
  switch (status) {
    case 'CHANGED':
      return 'gold';
    case 'NEW':
      return 'cyan';
    case 'REMOVED':
      return 'red';
    case 'UNCHANGED':
      return undefined;
  }
};

const ComparisonCard = ({ pair }: { pair: PairedImageReleaseContext }) => (
  <Card
    title={
      <Flex justify='space-between' align='center' gap='small' wrap>
        <Typography.Text className='whitespace-normal break-all'>{imageName(pair)}</Typography.Text>
        <Tag color={statusColor(pair.status)}>{pair.status}</Tag>
      </Flex>
    }
  >
    <Row gutter={[24, 24]}>
      <Col xs={24} xl={12}>
        <Typography.Text type='secondary' strong className='block mb-3'>
          Current
        </Typography.Text>
        <ImageDetails context={pair.current} />
      </Col>
      <Col xs={24} xl={12}>
        <Typography.Text type='secondary' strong className='block mb-3'>
          Incoming
        </Typography.Text>
        <ImageDetails context={pair.incoming} />
      </Col>
    </Row>
  </Card>
);

export const ReleaseContext = ({ freight, currentFreight, comparison }: ReleaseContextProps) => {
  const project = freight.metadata?.namespace || '';
  const freightName = freight.metadata?.name || '';
  const config = useGetFreightReleaseContextConfig(project, freightName, {
    query: { enabled: !!project && !!freightName, refetchOnWindowFocus: true }
  });
  const mappings = config.isError ? undefined : config.data?.data.imageAnnotations;
  const configStatus = config.isLoading ? (
    <Skeleton active title={false} paragraph={{ rows: 1 }} />
  ) : config.isError ? (
    <Alert
      type='warning'
      showIcon
      message='Custom annotation configuration could not be loaded'
      description='Only standard OCI fields are interpreted. Raw image annotations remain available.'
      action={
        <Button size='small' onClick={() => config.refetch()}>
          Retry
        </Button>
      }
    />
  ) : null;

  if (comparison) {
    const pairs = pairImageReleaseContexts(currentFreight, freight, mappings);
    return (
      <Space direction='vertical' size='large' className='w-full'>
        <Typography.Title level={4}>Container image changes</Typography.Title>
        {configStatus}
        {pairs.length > 0 ? (
          pairs.map((pair) => <ComparisonCard key={pair.key} pair={pair} />)
        ) : (
          <Empty description='Neither Freight resource contains container images.' />
        )}
      </Space>
    );
  }

  const contexts = getFreightImageReleaseContexts(freight, mappings);
  return (
    <Space direction='vertical' size='large' className='w-full'>
      <Typography.Title level={4}>Container images in this Freight</Typography.Title>
      {configStatus}
      {contexts.length > 0 ? (
        contexts.map((context) => (
          <Card
            key={context.image.repoURL}
            title={
              <Typography.Text className='whitespace-normal break-all'>
                {context.image.repoURL || 'Container image'}
              </Typography.Text>
            }
          >
            <ImageDetails context={context} />
          </Card>
        ))
      ) : (
        <Empty description='This Freight contains no container images.' />
      )}
    </Space>
  );
};
