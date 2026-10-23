import { Empty, Flex } from 'antd';
import { useMemo } from 'react';

import { LoadingState } from '@ui/features/common/loading-state';
import { TARGETS_POLL_INTERVAL_MS } from '@ui/features/project/targets/constants';
import { freightNames } from '@ui/features/project/targets/model/round-utils';
import { targetRows } from '@ui/features/project/targets/model/target-rows';
import { useFreightLabel } from '@ui/features/project/targets/shared/use-freight-label';
import { useListTargets } from '@ui/gen/api/v2/core/core';
import { PromotionRollout, Stage } from '@ui/gen/api/v2/models';

import { freightMarking } from './marks';
import { TargetMatrix } from './target-matrix';
import { TargetTiles } from './tiles';
import { useMarkColors } from './use-mark-colors';

type Props = {
  projectName: string;
  // The Stage whose Targets are shown. Its name scopes the Target list and
  // keys each Target's per-Stage status; its current Freight collection is
  // what a Target is compared against to be called behind.
  stage: Stage;
  // The Stage's latest round, resolved once by the drawer, which also uses it
  // to explain a blocked round above the tabs. Passing it in keeps one
  // PromotionRollout fetch and watch per drawer rather than one per tab.
  round?: PromotionRollout;
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
 * Governance comes from the API's `stage` filter, which evaluates the Stage's
 * selectors exactly as the controller does. Freight and health per Target come
 * from the Target's own status for this Stage; the round's outcome comes from
 * the Stage's current PromotionRollout, which the drawer resolves once and
 * passes in.
 */
export const StageTargets = ({ projectName, stage, round }: Props) => {
  const stageName = stage.metadata?.name || '';

  // Targets live in the database, which offers no watch; the list is polled.
  const targetsQuery = useListTargets(
    projectName,
    { stage: stageName },
    { query: { refetchInterval: TARGETS_POLL_INTERVAL_MS } }
  );

  const freightLabel = useFreightLabel(projectName);
  const colors = useMarkColors();

  const targets = useMemo(() => targetsQuery.data?.data?.items || [], [targetsQuery.data]);
  // Rows are built against this Stage alone, so every state on the tab is the
  // Target as this Stage sees it, not as the fleet at large does.
  const rows = useMemo(
    () => targetRows(targets, [stage], { [stageName]: round }),
    [targets, stage, stageName, round]
  );

  const stageFreight = useMemo(() => freightNames(stage.status?.freightHistory?.[0]), [stage]);
  // Built once here rather than in each panel, so the Freight tile and the
  // grid's Freight colouring paint a piece of Freight the same colour.
  const freightMarks = useMemo(
    () => freightMarking(rows, stageFreight, colors, freightLabel),
    [rows, stageFreight, colors, freightLabel]
  );

  if (targetsQuery.isLoading) {
    return <LoadingState />;
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
        round={round}
        freightMarks={freightMarks}
        freightLabel={freightLabel}
      />
      <TargetMatrix
        project={projectName}
        stageName={stageName}
        rows={rows}
        round={round}
        stageFreight={stageFreight}
        freightLabel={freightLabel}
      />
    </Flex>
  );
};
