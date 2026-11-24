import { useContext, useMemo } from 'react';
import { generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ColorContext } from '@ui/context/colors';
import { ColorMapHex, parseColorAnnotation } from '@ui/features/stage/utils';
import { isStagePromoting, rolloutPath } from '@ui/features/stage/utils/promotion-rollout';
import { useGetPromotion } from '@ui/gen/api/v2/core/core';
import {
  PromotionReference,
  PromotionRequestReference,
  PromotionRollout,
  PromotionRolloutReference,
  Stage
} from '@ui/gen/api/v2/models';

import { IAction, useActionContext } from '../context/action-context';
import { useDictionaryContext } from '../context/dictionary-context';
import { useFreightTimelineControllerContext } from '../context/freight-timeline-controller-context';

export const isStageControlFlow = (stage: Stage) =>
  (stage?.spec?.promotionTemplate?.spec?.steps?.length || 0) <= 0;

// isStageTargetAware reports whether a Stage governs Targets, and therefore
// promotes Freight via a PromotionRollout or a PromotionRequest rather than a
// Promotion.
//
// This mirrors api.IsTargetAware on the back end: the targets block being
// present is the test, not what its selectors match. A Stage whose selectors
// currently match nothing is still target-aware.
export const isStageTargetAware = (stage?: Stage) => stage?.spec?.targets !== undefined;

export const getStageHealth = (stage: Stage) => stage?.status?.health;

export const useIsColorsUsed = () => {
  const freightTimelineControllerContext = useFreightTimelineControllerContext();

  return freightTimelineControllerContext?.preferredFilter?.showColors;
};

// StagePromotionRef is a Stage's account of what it is promoting now, or what
// it last promoted, in one shape regardless of how the Stage promotes.
//
// A classic Stage promotes through a Promotion and records it in
// status.currentPromotion and status.lastPromotion. A target-aware Stage
// promotes through a round that fans out one child Promotion per Target. Where
// that round is a PromotionRequest, the Stage records it in
// status.currentPromotionRequest and status.lastPromotionRequest. Where it is
// a PromotionRollout, the Stage records only the last one it saw finish, in
// status.lastPromotionRollout: a rollout lives in the database, and the one in
// flight is read from there. For such a Stage the Promotion fields name at
// most one of the children, so they never stand for the round.
export type StagePromotionRef = {
  kind: 'Promotion' | 'PromotionRequest' | 'PromotionRollout';
  name?: string;
  phase?: string;
  message?: string;
  finishedAt?: string;
  freightName?: string;
  freightOrigin?: string;
  // path is where the UI shows the referenced promotion activity. A Promotion
  // has a page of its own, and so does a PromotionRollout, inside its Stage's
  // Promotions tab. A PromotionRequest is listed on the Stage page, whose
  // Promotions tab leads with the Stage's PromotionRequests.
  path?: string;
};

const promotionPath = (stage?: Stage, promotion?: string) =>
  promotion
    ? generatePath(paths.promotion, {
        name: stage?.metadata?.namespace || '',
        promotionId: promotion
      })
    : undefined;

const stagePath = (stage?: Stage) =>
  stage?.metadata?.namespace && stage?.metadata?.name
    ? generatePath(paths.stage, {
        name: stage.metadata.namespace,
        stageName: stage.metadata.name
      })
    : undefined;

const promotionRef = (stage?: Stage, ref?: PromotionReference): StagePromotionRef | undefined =>
  ref && {
    kind: 'Promotion',
    name: ref.name,
    phase: ref.status?.phase,
    message: ref.status?.message,
    finishedAt: ref.finishedAt,
    freightName: ref.freight?.name,
    freightOrigin: ref.freight?.origin?.name,
    path: promotionPath(stage, ref.name)
  };

const promotionRequestRef = (
  stage?: Stage,
  ref?: PromotionRequestReference
): StagePromotionRef | undefined =>
  ref && {
    kind: 'PromotionRequest',
    name: ref.name,
    phase: ref.phase,
    finishedAt: ref.finishedAt,
    freightName: ref.freight?.name,
    freightOrigin: ref.freight?.origin?.name,
    path: stagePath(stage)
  };

// promotionRolloutRef describes a PromotionRollout. One with a number leads to
// its own page; one that is known of but not yet read leads to the Stage.
const promotionRolloutRef = (
  stage?: Stage,
  ref?: PromotionRolloutReference,
  freightName?: string
): StagePromotionRef | undefined =>
  ref && {
    kind: 'PromotionRollout',
    name: ref.name,
    phase: ref.phase,
    message: ref.message,
    finishedAt: ref.finishedAt,
    freightName,
    path:
      ref.number && stage?.metadata?.namespace && stage?.metadata?.name
        ? rolloutPath(stage.metadata.namespace, stage.metadata.name, ref.number)
        : stagePath(stage)
  };

// getCurrentPromotionRef describes what the Stage is promoting right now, if
// anything.
//
// A Stage names the Promotion or PromotionRequest it is promoting through,
// but not the PromotionRollout: unfinished is the Stage's rollout in flight as
// the list of rollouts has it, for callers that read that list. A Stage that
// fans out through rollouts and says it is promoting, with no rollout to show
// for it yet, is still promoting -- through a rollout that has not been read.
export const getCurrentPromotionRef = (
  stage?: Stage,
  unfinished?: PromotionRollout
): StagePromotionRef | undefined => {
  if (!isStageTargetAware(stage)) {
    return promotionRef(stage, stage?.status?.currentPromotion);
  }
  if (unfinished) {
    return promotionRolloutRef(
      stage,
      {
        name: unfinished.metadata?.name,
        number: unfinished.number,
        phase: unfinished.status?.phase,
        message: unfinished.status?.message
      },
      unfinished.spec?.freight
    );
  }
  const request = promotionRequestRef(stage, stage?.status?.currentPromotionRequest);
  if (request) {
    return request;
  }
  return stage?.status?.lastPromotionRollout && isStagePromoting(stage)
    ? promotionRolloutRef(stage, {})
    : undefined;
};

// getLastPromotionRef describes the last promotion the Stage saw through to a
// terminal phase, if any. A Stage that fans out through PromotionRollouts may
// still carry the reference of a PromotionRequest from before it did, so the
// rollout comes first.
export const getLastPromotionRef = (stage?: Stage): StagePromotionRef | undefined => {
  if (!isStageTargetAware(stage)) {
    return promotionRef(stage, stage?.status?.lastPromotion);
  }
  return (
    promotionRolloutRef(stage, stage?.status?.lastPromotionRollout) ||
    promotionRequestRef(stage, stage?.status?.lastPromotionRequest)
  );
};

export const getLastPromotionDate = (stage: Stage) => {
  const finishedAt = getLastPromotionRef(stage)?.finishedAt;
  return finishedAt ? new Date(finishedAt) : null;
};

// getCurrentPromotion names the Promotion a Stage is currently promoting
// through, for callers that need to read the Promotion itself. A target-aware
// Stage promotes through a round, whose child Promotions target distinct
// Targets and run in parallel, so no single Promotion describes what it is
// doing: for such a Stage this is always undefined.
export const getCurrentPromotion = (stage: Stage) => {
  const current = getCurrentPromotionRef(stage);
  return current?.kind === 'Promotion' ? current.name : undefined;
};

export const useCurrentPromotion = (stage: Stage) => {
  const currentPromotion = getCurrentPromotion(stage);

  const query = useGetPromotion(stage?.metadata?.namespace || '', currentPromotion || '', {
    query: {
      enabled: !!currentPromotion,
      staleTime: 10 * 1000,
      gcTime: 10 * 1000
    }
  });

  return { promotion: query.data?.data, isFetching: query.isFetching };
};

export const useHideStageIfInPromotionMode = (stage: Stage) => {
  const actionContext = useActionContext();
  const dictionaryContext = useDictionaryContext();

  return useMemo(() => {
    if (
      actionContext?.action?.type !== IAction.PROMOTE &&
      actionContext?.action?.type !== IAction.PROMOTE_DOWNSTREAM
    ) {
      return false;
    }

    const isSameStage = actionContext?.action?.stage?.metadata?.name === stage?.metadata?.name;

    if (isSameStage) {
      return false;
    }

    if (actionContext?.action?.type === IAction.PROMOTE) {
      const isParentStage = actionContext?.action?.stage?.spec?.requestedFreight?.find((f) =>
        f.sources?.stages?.includes(stage?.metadata?.name || '')
      );

      if (isParentStage) {
        return false;
      }

      return true;
    }

    if (
      dictionaryContext?.subscribersByStage?.[
        actionContext?.action?.stage?.metadata?.name || ''
      ]?.has(stage?.metadata?.name || '')
    ) {
      return false;
    }

    return true;
  }, [stage, actionContext?.action, dictionaryContext?.subscribersByStage]);
};

// useStageAccentColor resolves the color a Stage is identified by in the
// pipeline graph, or undefined when the user has turned colors off.
export const useStageAccentColor = (stage: Stage): string | undefined => {
  const colorContext = useContext(ColorContext);
  const colorsUsed = useIsColorsUsed();

  if (!colorsUsed) {
    return undefined;
  }

  const stageColor =
    parseColorAnnotation(stage) || colorContext.stageColorMap[stage?.metadata?.name || ''];

  return ColorMapHex[stageColor] || stageColor || undefined;
};
