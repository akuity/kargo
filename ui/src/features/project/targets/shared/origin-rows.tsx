import { Flex, Tag, Tooltip, Typography } from 'antd';
import { ReactNode } from 'react';

import { OriginFreight } from '../model/round-utils';
import { artifactSummary, referenceArtifacts } from '../model/target-details-utils';

import { FreightLink } from './freight-link';

// Row is one labelled value in a card: the label on the left, the value on the
// right, as the Stage and Freight detail panels lay theirs out.
export const Row = ({ label, children }: { label: string; children: ReactNode }) => (
  <Flex justify='space-between' align='flex-start' gap={12}>
    <span className='text-xs text-gray-500 dark:text-neutral-400 shrink-0'>{label}</span>
    <span className='text-xs text-right min-w-0 break-words'>{children}</span>
  </Flex>
);

/**
 * OriginRows is a Target's Freight laid out one origin to a line: what the
 * origin's Freight carries, the Freight itself, and what the Stage has moved
 * that origin on to when it has. A Stage requesting from many Warehouses puts
 * as many pieces of Freight on a Target, and listed apart from their artifacts
 * and from the Stage's own, nothing says which goes with which.
 */
export const OriginRows = ({
  project,
  origins,
  freightLabel
}: {
  project: string;
  origins: OriginFreight[];
  freightLabel: (name: string) => string;
}) => (
  <>
    {origins.map(({ origin, current, stage, behind }) => (
      <Row key={origin} label={origin}>
        <Flex gap={8} wrap justify='flex-end' align='center'>
          {/* Shortened names can repeat across registries, so the index keeps
              each key unique. */}
          {referenceArtifacts(current).map(({ kind, text }, i) => (
            <span key={`${kind}:${text}:${i}`} className='font-mono' title={kind}>
              {text}
            </span>
          ))}
          {current?.name ? (
            <FreightLink
              project={project}
              name={current.name}
              label={freightLabel(current.name)}
              className='text-gray-500 dark:text-neutral-400'
              showName={false}
            />
          ) : (
            <Typography.Text type='secondary' className='text-xs'>
              none
            </Typography.Text>
          )}
          {behind && stage?.name && (
            <Tooltip title={`Stage is on ${artifactSummary(stage) || 'other Freight'}`}>
              <Tag variant='filled' color='warning' className='m-0 text-xs'>
                behind
              </Tag>
            </Tooltip>
          )}
        </Flex>
      </Row>
    ))}
  </>
);
