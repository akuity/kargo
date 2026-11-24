import { isStageTargetAware } from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { Stage, Target } from '@ui/gen/api/v2/models';

import { matchesLabelSelector } from './target-labels';

// stageGovernsTarget mirrors the controller's rule: a Stage governs a Target
// when the Target matches any of the Stage's selectors. A classic Stage governs
// none, and so does a target-aware Stage whose selector list is empty.
export const stageGovernsTarget = (stage: Stage, target: Target): boolean =>
  isStageTargetAware(stage) &&
  (stage.spec?.targets?.selectors || []).some((selector) =>
    matchesLabelSelector(selector, target.metadata?.labels || {})
  );

export const byName = <T extends { metadata?: { name?: string } }>(lhs: T, rhs: T) =>
  (lhs.metadata?.name || '').localeCompare(rhs.metadata?.name || '');

// targetAwareStages returns the Stages that promote through Targets, by name.
export const targetAwareStages = (stages: Stage[]): Stage[] =>
  stages.filter((stage) => isStageTargetAware(stage)).sort(byName);

// stageRanks gives each Stage its column in the pipeline: one past the deepest
// Stage it requests Freight from. Same edges the pipeline view's graph uses, so
// the two agree without laying a graph out. Runs over every Stage, since a
// classic one upstream still pushes what follows it along.
const stageRanks = (stages: Stage[]): Map<string, number> => {
  const byName = new Map(stages.map((stage) => [stage.metadata?.name || '', stage]));
  const ranks = new Map<string, number>();
  const visiting = new Set<string>();

  const rankOf = (name: string): number => {
    const known = ranks.get(name);
    if (known !== undefined) {
      return known;
    }
    // A pipeline is a DAG, but a malformed one must not hang the page: a Stage
    // met again mid-walk is treated as a root rather than recurred into.
    if (visiting.has(name)) {
      return 0;
    }
    visiting.add(name);
    const upstream = (byName.get(name)?.spec?.requestedFreight || []).flatMap(
      (requested) => requested.sources?.stages || []
    );
    const rank = upstream.length ? Math.max(...upstream.map(rankOf)) + 1 : 0;
    visiting.delete(name);
    ranks.set(name, rank);
    return rank;
  };

  for (const stage of stages) {
    rankOf(stage.metadata?.name || '');
  }
  return ranks;
};

// pipelineOrder sorts Stages upstream first, as the pipeline view lays them
// out. One rank sorts by name, which at least stays stable between renders.
export const pipelineOrder = (stages: Stage[], all: Stage[] = stages): Stage[] => {
  const ranks = stageRanks(all);
  const rankOf = (stage: Stage) => ranks.get(stage.metadata?.name || '') ?? 0;
  return [...stages].sort((lhs, rhs) => rankOf(lhs) - rankOf(rhs) || byName(lhs, rhs));
};
