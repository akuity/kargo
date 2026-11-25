import { Empty, Flex } from 'antd';
import { useMemo } from 'react';

import { LoadingState } from '@ui/features/common/loading-state';
import { TARGETS_POLL_INTERVAL_MS } from '@ui/features/project/targets/constants';
import { freightNames } from '@ui/features/project/targets/model/target-freight';
import { targetRows } from '@ui/features/project/targets/model/target-rows';
import { stageFigures } from '@ui/features/project/targets/model/target-status';
import { StatusesError } from '@ui/features/project/targets/shared/statuses-error';
import {
  useFreightLabel,
  useFreightReference
} from '@ui/features/project/targets/shared/use-freight-label';
import { useTargetStatuses } from '@ui/features/project/targets/use-target-statuses';
import { Round } from '@ui/features/stage/utils/promotion-rollout';
import { useListTargets } from '@ui/gen/api/v2/core/core';
import { Stage } from '@ui/gen/api/v2/models';

import { freightMarking } from './marks';
import { TargetMatrix } from './target-matrix';
import { TargetTiles } from './tiles';
import { useMarkColors } from './use-mark-colors';

type Props = {
  projectName: string;
  // The Stage whose Targets are shown. Its name scopes the statuses; its
  // current Freight collection is what a Target is compared against, origin
  // by origin, to be called behind on one.
  stage: Stage;
  // The Stage's latest round, resolved once by the drawer, which also uses it
  // to explain a blocked round above the tabs. Here it is only linked to.
  round?: Round;
};

/**
 * StageTargets is a Stage's Targets tab: what the Stage governs and how that fleet is
 * doing, at the size a fleet is actually read -- hundreds of Targets at once,
 * as squares rather than as rows.
 *
 * It opens with a few figures for the fleet as a whole, then draws every
 * Target as a square, sliced along whichever label the reader picks. Picking a
 * square answers for that one Target, under the grid.
 *
 * Which Targets the Stage governs, and where each stands with it, comes from
 * the server's Target statuses for this Stage, derived from every Promotion
 * and PromotionRollout on record -- not only the latest round, so a Target a
 * retry left out keeps the state it had. The Targets themselves fill in their
 * params, owner and any recorded health.
 */
export const StageTargets = ({ projectName, stage, round }: Props) => {
  const stageName = stage.metadata?.name || '';
  const stages = useMemo(() => [stage], [stage]);

  // Targets live in the database, which offers no watch; the list is polled.
  const targetsQuery = useListTargets(
    projectName,
    { stage: stageName },
    { query: { refetchInterval: TARGETS_POLL_INTERVAL_MS } }
  );
  const targets = useMemo(() => targetsQuery.data?.data?.items || [], [targetsQuery.data]);
  const statuses = useTargetStatuses(
    projectName,
    stages,
    targetsQuery.isLoading ? undefined : targets,
    stageName
  );

  const freightLabel = useFreightLabel(projectName);
  const resolveFreight = useFreightReference(projectName);
  const colors = useMarkColors();

  // Rows are built from this Stage's statuses alone, so every state on the
  // tab is the Target as this Stage sees it, not as the fleet at large does.
  // A Target the list holds and the statuses do not -- one whose labels have
  // just changed -- waits for the statuses to catch up.
  const rows = useMemo(
    () =>
      targetRows(targets, statuses.entries || [], stages, resolveFreight).filter(
        (row) => row.stages.length > 0
      ),
    [targets, statuses.entries, stages, resolveFreight]
  );
  const figures = useMemo(
    () => stageFigures(statuses.summary, stageName),
    [statuses.summary, stageName]
  );

  const stageFreight = useMemo(() => freightNames(stage.status?.freightHistory?.[0]), [stage]);
  // Built once here rather than in each panel, so the Freight tile and the
  // grid's Freight colouring paint a piece of Freight the same colour.
  const freightMarks = useMemo(
    () => freightMarking(rows, stageFreight, colors, freightLabel),
    [rows, stageFreight, colors, freightLabel]
  );

  if (statuses.isLoading || targetsQuery.isLoading) {
    return <LoadingState />;
  }

  // Without statuses no Target would match, which says the selectors are
  // wrong when it is the read that failed.
  if (statuses.error) {
    return <StatusesError error={statuses.error} />;
  }

  if (rows.length === 0) {
    return (
      <Empty
        image={Empty.PRESENTED_IMAGE_SIMPLE}
        description="No Targets match this Stage's selectors"
      />
    );
  }

  return (
    <Flex vertical gap={16}>
      <TargetTiles
        project={projectName}
        stageName={stageName}
        rows={rows}
        figures={figures}
        round={round}
        stageFreight={stageFreight}
        freightMarks={freightMarks}
        freightLabel={freightLabel}
      />
      <TargetMatrix
        project={projectName}
        stageName={stageName}
        rows={rows}
        stageFreight={stageFreight}
        freightLabel={freightLabel}
      />
    </Flex>
  );
};
