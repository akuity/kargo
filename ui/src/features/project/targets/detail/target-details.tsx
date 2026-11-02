import { faFileCode, faRocket } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Tag, Tooltip, Typography } from 'antd';
import { formatDistanceToNow } from 'date-fns';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useModal } from '@ui/features/common/modal/use-modal';
import { SmallLabel } from '@ui/features/common/small-label';
import { parseDate } from '@ui/utils/dates';

import { OriginFreight, freightByOrigin } from '../model/round-utils';
import { artifactSummary, flattenParams, referenceArtifacts } from '../model/target-details-utils';
import { GoverningStage, TargetRow } from '../model/target-rows';
import { targetStatePresentations } from '../model/target-state';
import { FreightLink } from '../shared/freight-link';
import { StateDot } from '../shared/state-visuals';

import { TargetPromotionsModal } from './target-promotions';
import { TargetYamlModal } from './target-yaml-modal';

const cardClassName =
  'rounded-md p-3 border border-solid bg-gray-50 dark:bg-neutral-900 ' +
  'border-gray-200 dark:border-neutral-700';

// Row is one labelled value in a card: the label on the left, the value on the
// right, as the Stage and Freight detail panels lay theirs out.
const Row = ({ label, children }: { label: string; children: React.ReactNode }) => (
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
const OriginRows = ({
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

/**
 * StageCard is what one Stage has put on a Target and what has become of it:
 * the Freight the Stage landed here, what the Stage itself has moved on to when
 * the two differ, the artifacts that Freight carries, and the Target's health
 * under that Stage, one line per origin the Freight comes from.
 */
const StageCard = ({
  project,
  governing,
  freightLabel
}: {
  project: string;
  governing: GoverningStage;
  freightLabel: (name: string) => string;
}) => {
  const stageName = governing.stage.metadata?.name || '';
  // A Target with no Freight is only shown what the Stage is on once a
  // promotion to it has failed. Until then it has simply not been promoted to,
  // and is not behind on anything.
  const origins =
    governing.currentFreight || governing.upToDate === false
      ? freightByOrigin(governing.currentFreight, governing.stage.status?.freightHistory?.[0])
      : [];
  const anyBehind = origins.some(({ behind }) => behind);
  const issues = governing.health?.issues || [];

  return (
    <div className={cardClassName}>
      <Flex justify='space-between' align='center' gap={8} className='mb-2'>
        <Typography.Text strong className='text-sm'>
          In <Link to={generatePath(paths.stage, { name: project, stageName })}>{stageName}</Link>
        </Typography.Text>
        <Flex gap={6} align='center'>
          <StateDot state={governing.state} />
          <span className='text-xs'>{targetStatePresentations[governing.state].label}</span>
        </Flex>
      </Flex>

      <Flex vertical gap={6}>
        {origins.length ? (
          <OriginRows project={project} origins={origins} freightLabel={freightLabel} />
        ) : (
          <Row label='Freight'>
            <Typography.Text type='secondary' className='text-xs'>
              none
            </Typography.Text>
          </Row>
        )}
        <Row label='Health'>
          {governing.health?.status ? (
            governing.health.status
          ) : (
            <Typography.Text type='secondary' className='text-xs'>
              not assessed
            </Typography.Text>
          )}
        </Row>
      </Flex>

      {/* Each origin's own line says when it is behind. A Stage that has moved
          on with no origin to show for it still needs saying somewhere. */}
      {governing.upToDate === false && !anyBehind && (
        <Typography.Text type='warning' className='text-xs block mt-2'>
          Behind: Stage is on other Freight
        </Typography.Text>
      )}
      {issues.length > 0 && (
        <Typography.Text type='danger' className='text-xs block mt-2'>
          {issues.join('; ')}
        </Typography.Text>
      )}
    </div>
  );
};

/**
 * TargetDetails is everything about one Target that does not fit on its row:
 * a card per Stage that governs it, and the params its promotion steps read.
 */
export const TargetDetails = ({
  project,
  row,
  freightLabel
}: {
  project: string;
  row: TargetRow;
  freightLabel: (name: string) => string;
}) => {
  const params = flattenParams(row.target.spec?.params);
  const created = parseDate(row.target.metadata?.creationTimestamp);
  const name = row.target.metadata?.name || '';
  const { show: showYaml } = useModal((props) => (
    <TargetYamlModal target={row.target} {...props} />
  ));
  const { show: showPromotions } = useModal((props) => (
    <TargetPromotionsModal project={project} target={name} {...props} />
  ));

  return (
    <Flex vertical gap={8}>
      <Flex justify='space-between' align='center' gap={12} wrap>
        <SmallLabel>
          {created
            ? `Created ${formatDistanceToNow(created, { addSuffix: true })} · ` +
              created.toLocaleString()
            : ''}
        </SmallLabel>
        <Flex gap={8}>
          <Button size='small' onClick={() => showYaml()}>
            <FontAwesomeIcon icon={faFileCode} className='mr-1' />
            YAML
          </Button>
          <Button size='small' onClick={() => showPromotions()}>
            <FontAwesomeIcon icon={faRocket} className='mr-1' />
            Promotions
          </Button>
        </Flex>
      </Flex>
      <div
        className='grid gap-3 items-start'
        style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))' }}
      >
        {row.stages.map((governing) => (
          <StageCard
            key={governing.stage.metadata?.name}
            project={project}
            governing={governing}
            freightLabel={freightLabel}
          />
        ))}
        {params.length > 0 && (
          <div className={cardClassName}>
            <Typography.Text strong className='text-sm block mb-2'>
              Params
            </Typography.Text>
            <Flex vertical gap={6}>
              {params.map(({ key, value }) => (
                <Row key={key} label={key}>
                  <span className='font-mono'>{value}</span>
                </Row>
              ))}
            </Flex>
          </div>
        )}
        {!row.stages.length && !params.length && (
          <Typography.Text type='secondary' className='text-xs'>
            No Stage selects this Target, and it carries no params.
          </Typography.Text>
        )}
      </div>
    </Flex>
  );
};
