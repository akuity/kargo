import { Flex, Segmented, Typography } from 'antd';
import { useMemo, useState } from 'react';

import {
  NO_GROUPING,
  TargetGroup,
  groupTargetRows,
  groupTitle
} from '@ui/features/project/targets/model/target-grouping';
import { labelKeys } from '@ui/features/project/targets/model/target-labels';
import { TargetRow } from '@ui/features/project/targets/model/target-rows';
import { surfaceClassName } from '@ui/features/project/targets/shared/surface';
import { TargetGrid } from '@ui/features/project/targets/summary/target-grid';
import { roundBlock } from '@ui/features/stage/utils/promotion-request';
import { PromotionRequest } from '@ui/gen/api/v2/models';
import { useLocalStorage } from '@ui/utils/use-local-storage';

import { MarkLegend } from './mark-visuals';
import { ColorMode, Marking, attentionTally, byMark, colorModes, marking, tally } from './marks';
import { assessedHealth } from './summary';
import { TargetFacts } from './target-facts';
import { useMarkColors } from './use-mark-colors';

// One square per Target at this size reads as a fleet rather than as a chart,
// and a group of a few hundred still fits the width of the drawer.
const squareSize = 12;

// Where a reader's choice of colouring and of grouping is kept between visits.
// The colouring is one preference across every Stage; the grouping is per
// project, because label keys are a project's own.
const colorKey = 'stage-targets-color';
const groupByKey = (project: string) => `stage-targets-group-by/${project}`;

// GroupRow is one slice of the fleet: what it is, how big it is, what is wrong
// with it, and a square per Target.
const GroupRow = ({
  group,
  groupKey,
  marks,
  picked,
  onPick
}: {
  group: TargetGroup;
  groupKey?: string;
  marks: Marking;
  // The name of the Target whose square is picked, wherever it sits.
  picked?: string;
  onPick: (row: TargetRow) => void;
}) => {
  const sorted = useMemo(() => [...group.rows].sort(byMark(marks)), [group.rows, marks]);
  const attention = useMemo(() => attentionTally(group.rows, marks), [group.rows, marks]);

  return (
    <div
      className='flex w-full gap-4 items-start py-3 border-0 border-t border-solid
        border-gray-100 dark:border-neutral-700'
    >
      <div className='w-[150px] shrink-0'>
        <div className='text-xs font-mono'>{groupTitle(group, groupKey)}</div>
        <Typography.Text type='secondary' className='block text-xs'>
          {group.rows.length} target{group.rows.length === 1 ? '' : 's'}
        </Typography.Text>
        {attention && (
          <div className='text-xs' style={{ color: attention.mark.color }}>
            {attention.count} {attention.mark.short || attention.mark.label}
          </div>
        )}
      </div>
      <div className='flex-1 min-w-0'>
        <TargetGrid
          rows={sorted}
          mark={marks.mark}
          size={squareSize}
          selected={picked}
          onSelect={onPick}
        />
      </div>
    </div>
  );
};

/**
 * TargetMatrix draws the whole fleet as squares, sliced into rows along
 * whichever label the reader picks -- tier, cloud, region, whatever the
 * Targets happen to carry -- and coloured for whichever reading they are after:
 * how the Stage's last round went, what Freight each Target runs, or how
 * healthy each is. Picking a square answers for the Target behind it.
 */
export const TargetMatrix = ({
  project,
  stageName,
  rows,
  round,
  stageFreight,
  freightLabel
}: {
  project: string;
  stageName: string;
  rows: TargetRow[];
  // The Stage's latest round, which a picked Target reads its failure from.
  round?: PromotionRequest;
  // The Freight the Stage itself is on: what every Target it governs is on its
  // way to, and the first colour of the Freight legend.
  stageFreight: string[];
  freightLabel: (name: string) => string;
}) => {
  const colors = useMarkColors();

  // Health is only a reading once something has assessed it. Until then it
  // would paint the whole fleet one shade of Unknown, which says less than not
  // offering it at all; it returns by itself the day health is recorded.
  const modes = useMemo(
    () => colorModes.filter(({ value }) => value !== 'health' || assessedHealth(rows)),
    [rows]
  );
  const [storedMode, setMode] = useLocalStorage<ColorMode>(colorKey);
  // Anything may be in storage, including a mode that is not on offer here.
  const mode = modes.some(({ value }) => value === storedMode) ? storedMode : 'promotion';

  const keys = useMemo(() => labelKeys(rows.map(({ target }) => target)), [rows]);
  const [storedGroupBy, setGroupBy] = useLocalStorage<string>(groupByKey(project));
  // Not grouping is the default, as it is on the project's Targets page: a
  // key picked for the reader could be one with a value per Target, which
  // would draw a row per Target instead of a fleet. A remembered key these
  // Targets do not carry falls back the same way.
  const groupBy =
    storedGroupBy === NO_GROUPING || keys.includes(storedGroupBy) ? storedGroupBy : NO_GROUPING;
  const groupKey = groupBy || undefined;

  // A blocked round reached no Target, so the squares say that rather than
  // reporting each Target as one the round never named.
  const blocked = !!roundBlock(round);
  const marks = useMemo(
    () => marking(mode, rows, stageFreight, colors, freightLabel, blocked),
    [mode, rows, stageFreight, colors, freightLabel, blocked]
  );
  const groups = useMemo(() => groupTargetRows(rows, groupKey), [rows, groupKey]);
  const tallies = useMemo(() => tally(rows, marks), [rows, marks]);

  // The picked Target is held by name, so the facts below the grid follow the
  // Target as its row is rebuilt -- a promotion landing on it while it is
  // being read updates the panel rather than closing it.
  const [pickedName, setPickedName] = useState<string>();
  const picked = rows.find((row) => row.target.name === pickedName);

  return (
    <div className={`px-4 py-3 ${surfaceClassName}`}>
      <Flex justify='space-between' align='center' gap={16} wrap>
        <Typography.Text strong>Targets</Typography.Text>
        <Flex gap={16} align='center' wrap>
          <Flex gap={8} align='center'>
            <Typography.Text type='secondary' className='text-xs'>
              Color
            </Typography.Text>
            <Segmented
              size='small'
              value={mode}
              options={modes}
              onChange={(value) => setMode(value as ColorMode)}
            />
          </Flex>
          {keys.length > 0 && (
            <Flex gap={8} align='center'>
              <Typography.Text type='secondary' className='text-xs'>
                Rows
              </Typography.Text>
              <Segmented
                size='small'
                value={groupBy}
                options={[
                  ...keys.map((key) => ({ value: key, label: key })),
                  { value: NO_GROUPING, label: 'none' }
                ]}
                onChange={(value) => setGroupBy(String(value))}
              />
            </Flex>
          )}
        </Flex>
      </Flex>

      <div className='my-3'>
        <MarkLegend tallies={tallies} />
      </div>

      {groups.map((group) => (
        <GroupRow
          key={group.value || 'all'}
          group={group}
          groupKey={groupKey}
          marks={marks}
          picked={pickedName}
          onPick={(row) =>
            setPickedName((current) => (current === row.target.name ? undefined : row.target.name))
          }
        />
      ))}

      {picked && (
        <TargetFacts
          project={project}
          stageName={stageName}
          row={picked}
          round={round}
          blocked={blocked}
          stageFreight={stageFreight}
          freightLabel={freightLabel}
          onClose={() => setPickedName(undefined)}
        />
      )}
    </div>
  );
};
