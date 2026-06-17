import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Col, Flex, Row, Space, Table, Tag, theme, Typography } from 'antd';
import { useMemo } from 'react';

import { ArtifactMetadata } from '@ui/features/freight/artifact-metadata';
import { Freight, FreightReference } from '@ui/gen/api/v1alpha1/generated_pb';

import { repoLabel, typeIcon, typeLabel, versionLabel } from './freight-comparison-utils';
import { PairedRow, pairArtifacts, PairStatus } from './pair-artifacts';

const statusTag = (status: PairStatus) => {
  switch (status) {
    case 'CHANGED':
      return <Tag color='gold'>CHANGED</Tag>;
    case 'NEW':
      return <Tag color='cyan'>NEW</Tag>;
    case 'REMOVED':
      return <Tag color='red'>REMOVED</Tag>;
    case 'UNCHANGED':
      return <Tag>UNCHANGED</Tag>;
  }
};

type FreightComparisonTableProps = {
  currentFreight?: Freight | FreightReference;
  incomingFreight: Freight;
  className?: string;
};

export const FreightComparisonTable = ({
  currentFreight,
  incomingFreight,
  className
}: FreightComparisonTableProps) => {
  const { token } = theme.useToken();
  const mutedColor = token.colorTextTertiary;
  const changedRowBg = `color-mix(in srgb, ${token.colorWarningBg} 50%, ${token.colorBgContainer})`;

  const rows = useMemo(
    () => pairArtifacts(currentFreight, incomingFreight),
    [currentFreight, incomingFreight]
  );

  const hasCurrent = !!currentFreight;

  return (
    <Table<PairedRow>
      className={className}
      dataSource={rows}
      rowKey='key'
      pagination={false}
      tableLayout='fixed'
      rowHoverable={false}
      onRow={(row) => {
        const expandable =
          (row.incoming || row.current)?.type === 'image' ||
          (row.incoming || row.current)?.type === 'git';
        const bg = row.status === 'CHANGED' || row.status === 'NEW' ? changedRowBg : undefined;
        return {
          style: {
            backgroundColor: bg,
            cursor: expandable ? 'pointer' : undefined
          }
        };
      }}
      expandable={{
        expandRowByClick: true,
        rowExpandable: (row) => {
          const t = (row.incoming || row.current)?.type;
          return t === 'image' || t === 'git';
        },
        expandedRowRender: (row) => (
          <Row gutter={24}>
            <Col span={12}>
              <Typography.Text type='secondary' strong className='text-xs uppercase mb-2 block'>
                Current
              </Typography.Text>
              {row.current ? (
                <ArtifactMetadata {...row.current} />
              ) : (
                <Typography.Text type='secondary'>—</Typography.Text>
              )}
            </Col>
            <Col span={12}>
              <Typography.Text type='secondary' strong className='text-xs uppercase mb-2 block'>
                Promoting
              </Typography.Text>
              {row.incoming ? (
                <ArtifactMetadata {...row.incoming} />
              ) : (
                <Typography.Text type='secondary'>—</Typography.Text>
              )}
            </Col>
          </Row>
        )
      }}
    >
      <Table.Column<PairedRow>
        title='Repo / Name'
        width='57%'
        render={(_, row) => {
          const source = row.incoming || row.current;
          if (!source) {
            return null;
          }
          return (
            <Flex align='start' gap={12}>
              <FontAwesomeIcon
                icon={typeIcon(source)}
                style={{
                  opacity: row.status === 'UNCHANGED' ? 0.55 : 1,
                  marginTop: 4
                }}
              />
              <div>
                <div className='font-mono'>{repoLabel(source)}</div>
                <Typography.Text type='secondary' className='text-xs'>
                  {typeLabel(source)}
                </Typography.Text>
              </div>
            </Flex>
          );
        }}
      />
      <Table.Column<PairedRow>
        title='Current'
        width='15%'
        render={(_, row) => {
          if (!hasCurrent) {
            return (
              <Typography.Text italic type='secondary'>
                — No freight —
              </Typography.Text>
            );
          }
          if (!row.current) {
            return <Typography.Text type='secondary'>—</Typography.Text>;
          }
          const muted = row.status === 'UNCHANGED';
          return (
            <span className='font-mono break-all' style={{ color: muted ? mutedColor : undefined }}>
              {versionLabel(row.current)}
            </span>
          );
        }}
      />
      <Table.Column<PairedRow>
        title='Promoting'
        width='18%'
        render={(_, row) => {
          if (!row.incoming) {
            return <Typography.Text type='secondary'>—</Typography.Text>;
          }
          if (row.status === 'UNCHANGED') {
            return (
              <span
                className='font-mono break-all inline-block w-4/5'
                style={{ color: mutedColor }}
              >
                {versionLabel(row.incoming)}
              </span>
            );
          }
          return (
            <Space size={4} align='center'>
              <span style={{ color: mutedColor }}>→</span>
              <Tag color='gold' className='font-mono whitespace-normal break-all'>
                {versionLabel(row.incoming)}
              </Tag>
            </Space>
          );
        }}
      />
      <Table.Column<PairedRow>
        title='Status'
        width='10%'
        render={(_, row) => statusTag(row.status)}
      />
    </Table>
  );
};
