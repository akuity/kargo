import { generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import {
  PromotionStatusPhase,
  isPromotionPhaseTerminal
} from '@ui/features/common/promotion-status/utils';
import {
  StageConditionStatus,
  StageConditionType,
  hasCondition
} from '@ui/features/common/stage-status/utils';
import { PromotionRequest, PromotionRollout, Stage } from '@ui/gen/api/v2/models';

// Round is one fan-out of a Stage's promotion across the Targets it governs,
// in the one shape every round view takes: the list of rounds, a round's
// detail, the Targets tab, the progress bar.
//
// A Stage fans out through a PromotionRollout, or through a PromotionRequest
// where rollouts are not in use. The two describe a round with the same
// fields, so a PromotionRequest is a Round as it stands and a PromotionRollout
// becomes one through rolloutToRound, which marks it by kind and keeps its
// number: a rollout is read back by its number, where a request is read by
// its name.
export type Round = PromotionRequest & { number?: number };

const promotionRolloutKind = 'PromotionRollout';

// rolloutToRound presents a PromotionRollout as a Round. It copies the fields
// a round is read by and nothing else, so a field a rollout gains later is
// simply not shown until a view asks for it.
export const rolloutToRound = (rollout: PromotionRollout): Round => ({
  kind: promotionRolloutKind,
  metadata: rollout.metadata,
  number: rollout.number,
  spec: {
    freight: rollout.spec?.freight || '',
    stage: rollout.spec?.stage || '',
    targets: (rollout.spec?.targets || []).flatMap(({ name }) => (name ? [{ name }] : [])),
    updateStrategy: rollout.spec?.updateStrategy
  },
  status: rollout.status && {
    conditions: rollout.status.conditions,
    phase: rollout.status.phase,
    message: rollout.status.message,
    startedAt: rollout.status.startedAt,
    finishedAt: rollout.status.finishedAt,
    summary: rollout.status.summary,
    targets: rollout.status.targets?.flatMap(({ name, ...rest }) =>
      name ? [{ name, ...rest }] : []
    )
  }
});

// isRolloutRound tells a round that is a PromotionRollout from one that is a
// PromotionRequest. A rollout without a number cannot be read back, so it does
// not count.
export const isRolloutRound = (round?: Round): round is Round & { number: number } =>
  round?.kind === promotionRolloutKind && typeof round.number === 'number';

// RoundIdentity is what a round is known by where it is listed or opened: a
// short label, and a hint for whoever wants more than the label says.
export type RoundIdentity = { label: string; hint?: string };

// roundIdentity names a round the way a reader would refer to it. A rollout
// is known by its number among its Stage's rollouts, which is also what it is
// read back by; its name identifies nothing a reader can follow, so it is
// kept as a hint, for matching the rollout to a Stage's status or a log line.
// A PromotionRequest has only its name.
export const roundIdentity = (round?: Round): RoundIdentity =>
  isRolloutRound(round)
    ? { label: `#${round.number}`, hint: round.metadata?.name }
    : { label: round?.metadata?.name || '' };

// roundMessage is what a round has to say for itself beyond its phase and its
// tally of Targets. A rollout that succeeded has nothing to add: its message
// only counts the Promotions that succeeded, which the tally beside it already
// does. Any other rollout's message says what it is waiting on or why it did
// not succeed, and a PromotionRequest's is shown whatever its phase.
export const roundMessage = (round?: Round): string | undefined => {
  const message = round?.status?.message || undefined;
  return isRolloutRound(round) && round.status?.phase === 'Succeeded' ? undefined : message;
};

export const rolloutPath = (project: string, stageName: string, number: number) =>
  generatePath(paths.promotionRollout, {
    name: project,
    stageName,
    promotionRolloutNumber: String(number)
  });

// roundPath is where a round is read: a page of its own inside the Stage's
// Promotions tab, addressed by number for a rollout and by name for a request.
// A round that has neither has nowhere to lead.
export const roundPath = (
  project: string,
  stageName: string,
  round?: Round
): string | undefined => {
  if (isRolloutRound(round)) {
    return rolloutPath(project, stageName, round.number);
  }
  const name = round?.metadata?.name;
  return name
    ? generatePath(paths.promotionRequest, { name: project, stageName, promotionRequestName: name })
    : undefined;
};

// promotedRoundPath is where to land after promoting to a Stage with Targets.
// The server answers with the round it created: a PromotionRollout, which
// carries a number, when it creates rollouts, and a PromotionRequest
// otherwise. The generated client types either as a Promotion, so the two are
// told apart by the number.
export const promotedRoundPath = (
  project: string,
  stageName: string,
  created?: { metadata?: { name?: string }; number?: number }
): string | undefined =>
  typeof created?.number === 'number'
    ? rolloutPath(project, stageName, created.number)
    : roundPath(project, stageName, created as Round | undefined);

// A rollout is finished once its phase is one it does not come back from. The
// phases are the Promotion's own, so the Promotion's test serves.
export const isRolloutFinished = (rollout?: PromotionRollout) =>
  isPromotionPhaseTerminal(rollout?.status?.phase as PromotionStatusPhase);

// rolloutCompareFn orders one Stage's PromotionRollouts newest first. A
// rollout's number is its position among its Stage's rollouts.
export const rolloutCompareFn = (lhs: PromotionRollout, rhs: PromotionRollout) =>
  (rhs.number || 0) - (lhs.number || 0);

// LatestRollouts is where one Stage stands: the rollout it is promoting
// through, if any, and the newest that has finished, if any.
export type LatestRollouts = {
  unfinished?: PromotionRollout;
  finished?: PromotionRollout;
};

// latestRolloutsByStage sorts a list of PromotionRollouts by Stage and keeps
// each Stage's latest. The list may be a whole Project's, and may hold more
// than the latest of each Stage, so it serves a page of history as well as
// the server's own list of latest rollouts.
export const latestRolloutsByStage = (
  rollouts: PromotionRollout[]
): Record<string, LatestRollouts> => {
  // A Stage may be named for anything an object inherits, "constructor" for
  // one, so the dictionary inherits nothing for such a name to collide with.
  const byStage: Record<string, LatestRollouts> = Object.create(null);
  for (const rollout of [...rollouts].sort(rolloutCompareFn)) {
    const stageName = rollout.spec?.stage;
    if (!stageName) {
      continue;
    }
    const latest = (byStage[stageName] ??= {});
    if (isRolloutFinished(rollout)) {
      latest.finished ??= rollout;
    } else {
      latest.unfinished ??= rollout;
    }
  }
  return byStage;
};

// stageUsesRollouts is the one rule deciding which of the two a Stage's views
// show. A Stage fans out through PromotionRollouts once it has recorded one,
// or once one exists for it -- the second covers a Stage's first rollout,
// which it has nothing to record of until it finishes. Any other Stage keeps
// its PromotionRequest views.
export const stageUsesRollouts = (stage: Stage | undefined, latest?: LatestRollouts): boolean =>
  !!stage?.status?.lastPromotionRollout || !!latest?.unfinished || !!latest?.finished;

// currentRollout is the rollout that best describes what a Stage is doing
// right now: the one in flight, else the newest that finished. A Stage names
// no current rollout of its own, so this is read off the list.
export const currentRollout = (latest?: LatestRollouts): PromotionRollout | undefined =>
  latest?.unfinished ?? latest?.finished;

// rolloutRound is the round of a Stage that fans out through
// PromotionRollouts, in the shape the round views take.
export const rolloutRound = (latest?: LatestRollouts): Round | undefined => {
  const rollout = currentRollout(latest);
  return rollout && rolloutToRound(rollout);
};

// rolloutRounds resolves that round for each of the Stages that fan out
// through rollouts, keyed by Stage name. A Stage that has recorded a rollout
// the list does not hold yet is among them, with no round to show for now;
// any other Stage is left out, for its PromotionRequests to speak for.
export const rolloutRounds = (
  stages: Stage[],
  byStage: Record<string, LatestRollouts>
): Record<string, Round | undefined> =>
  Object.fromEntries(
    stages
      .filter((stage) => stage.metadata?.name)
      .map((stage) => [stage.metadata?.name || '', stage] as const)
      .filter(([stageName, stage]) => stageUsesRollouts(stage, byStage[stageName]))
      .map(([stageName]) => [stageName, rolloutRound(byStage[stageName])])
  );

// isStagePromoting reports whether a Stage says it is promoting. For a Stage
// that fans out through rollouts this condition is the Stage's only word on a
// rollout in flight.
export const isStagePromoting = (stage?: Stage): boolean =>
  hasCondition(
    StageConditionType.Promoting,
    StageConditionStatus.True,
    stage?.status?.conditions || []
  ).isActive;

// shouldPollRollouts decides whether a list of rollouts can still change by
// itself. PromotionRollouts live in the database, which offers no watch, so a
// list is read again on a timer -- but only while something is in flight: a
// rollout in the list has yet to finish, or a Stage says it is promoting. The
// second covers a rollout the list has not shown yet. A Stage that names a
// PromotionRequest as current is promoting through that, not through a
// rollout, and is no reason to read the list.
export const shouldPollRollouts = (stages: Stage[], rollouts: PromotionRollout[]): boolean =>
  rollouts.some((rollout) => !isRolloutFinished(rollout)) ||
  stages.some((stage) => isStagePromoting(stage) && !stage.status?.currentPromotionRequest);

// recordedRollouts names the last rollout each Stage has recorded, as one
// string to compare. A rollout can start and finish between two reads of a
// list, or while nothing is reading it; the Stage, which is watched, then
// records a rollout the list has not caught up with. A change in this string
// is the sign that a list is due another read.
export const recordedRollouts = (stages: Stage[]): string =>
  stages
    .map((stage) => `${stage.metadata?.name}=${stage.status?.lastPromotionRollout?.name || ''}`)
    .sort()
    .join(',');
