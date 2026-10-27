import { faFileCode, faRocket } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Typography } from 'antd';
import { formatDistanceToNow } from 'date-fns';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useModal } from '@ui/features/common/modal/use-modal';
import { SmallLabel } from '@ui/features/common/small-label';
import { parseDate } from '@ui/utils/dates';

import { freightNames } from '../model/round-utils';
import { flattenParams, freightArtifacts } from '../model/target-details-utils';
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

const FreightLinks = ({
  project,
  names,
  freightLabel
}: {
  project: string;
  names: string[];
  freightLabel: (name: string) => string;
}) =>
  names.length ? (
    <Flex gap={6} wrap justify='flex-end'>
      {names.map((name) => (
        <FreightLink key={name} project={project} name={name} label={freightLabel(name)} />
      ))}
    </Flex>
  ) : (
    <Typography.Text type='secondary' className='text-xs'>
      none
    </Typography.Text>
  );

/**
 * StageCard is what one Stage has put on a Target and what has become of it:
 * the Freight the Stage landed here, what the Stage itself has moved on to when
 * the two differ, the artifacts that Freight carries, and the Target's health
 * under that Stage.
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
  const stageFreight = freightNames(governing.stage.status?.freightHistory?.[0]);
  const artifacts = freightArtifacts(governing.currentFreight);
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
        <Row label='Freight'>
          <FreightLinks
            project={project}
            names={freightNames(governing.currentFreight)}
            freightLabel={freightLabel}
          />
        </Row>
        {governing.upToDate === false && (
          <Row label='Stage Freight'>
            <FreightLinks project={project} names={stageFreight} freightLabel={freightLabel} />
          </Row>
        )}
        {artifacts.map(({ kind, text }) => (
          <Row key={`${kind}:${text}`} label={kind}>
            <span className='font-mono'>{text}</span>
          </Row>
        ))}
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

      {governing.upToDate === false && (
        <Typography.Text type='warning' className='text-xs block mt-2'>
          Behind: Stage is on {stageFreight.map(freightLabel).join(', ') || 'other Freight'}, Target
          on {freightNames(governing.currentFreight).map(freightLabel).join(', ') || 'none'}
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
