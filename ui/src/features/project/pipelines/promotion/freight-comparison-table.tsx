import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Table, Tag, theme, Typography } from 'antd';
import { useMemo } from 'react';

import { TableSource } from '@ui/features/freight/flatten-freight-origin-utils';
import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

import {
  relativeFromFreight,
  repoLabel,
  typeIcon,
  typeLabel,
  versionLabel
} from './freight-comparison-utils';
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

const ArtifactExtras = ({
  source,
  relativeAge,
  color
}: {
  source: TableSource;
  relativeAge: string;
  color?: string;
}) => {
  if (source.type !== 'git') {
    return null;
  }
  const author = [source.author, relativeAge].filter(Boolean).join(' · ');
  return (
    <div className='mt-1 text-xs leading-snug' style={{ color }}>
      {source.message && <div>{source.message}</div>}
      {author && <div>{author}</div>}
    </div>
  );
};

type FreightComparisonTableProps = {
  currentFreight?: Freight;
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
  const currentAge = useMemo(() => relativeFromFreight(currentFreight), [currentFreight]);
  const incomingAge = useMemo(() => relativeFromFreight(incomingFreight), [incomingFreight]);

  return (
    <Table<PairedRow>
      className={className}
      dataSource={rows}
      rowKey='key'
      pagination={false}
      onRow={(row) => ({
        style:
          row.status === 'CHANGED' || row.status === 'NEW'
            ? { backgroundColor: changedRowBg }
            : undefined
      })}
    >
      <Table.Column<PairedRow>
        title='Type'
        width={64}
        render={(_, row) => {
          const source = row.incoming || row.current;
          if (!source) {
            return null;
          }
          return (
            <FontAwesomeIcon
              icon={typeIcon(source)}
              style={{ opacity: row.status === 'UNCHANGED' ? 0.55 : 1 }}
            />
          );
        }}
      />
      <Table.Column<PairedRow>
        title='Repo / Name'
        render={(_, row) => {
          const source = row.incoming || row.current;
          if (!source) {
            return null;
          }
          return (
            <div>
              <div className='font-mono'>{repoLabel(source)}</div>
              <Typography.Text type='secondary' className='text-xs'>
                {typeLabel(source)}
              </Typography.Text>
            </div>
          );
        }}
      />
      <Table.Column<PairedRow>
        title='Current'
        render={(_, row) => {
          if (!hasCurrent) {
            return (
              <Typography.Text italic type='secondary'>
                — No current freight —
              </Typography.Text>
            );
          }
          if (!row.current) {
            return <Typography.Text type='secondary'>—</Typography.Text>;
          }
          const muted = row.status === 'UNCHANGED';
          const color = muted ? mutedColor : undefined;
          return (
            <div style={{ color }}>
              <span className='font-mono'>{versionLabel(row.current)}</span>
              <ArtifactExtras source={row.current} relativeAge={currentAge} color={color} />
            </div>
          );
        }}
      />
      <Table.Column<PairedRow>
        title='Promoting'
        render={(_, row) => {
          if (!row.incoming) {
            return <Typography.Text type='secondary'>—</Typography.Text>;
          }
          if (row.status === 'UNCHANGED') {
            return (
              <div style={{ color: mutedColor }}>
                <span className='mr-1'>=</span>
                <span className='font-mono'>{versionLabel(row.incoming)}</span>
              </div>
            );
          }
          return (
            <div>
              <span style={{ color: mutedColor }} className='mr-1'>
                →
              </span>
              <Tag color='gold' className='font-mono'>
                {versionLabel(row.incoming)}
              </Tag>
              <ArtifactExtras source={row.incoming} relativeAge={incomingAge} />
            </div>
          );
        }}
      />
      <Table.Column<PairedRow>
        title='Status'
        width={120}
        render={(_, row) => statusTag(row.status)}
      />
    </Table>
  );
};
