import { CSSProperties, useContext, useMemo } from 'react';
import { generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ColorContext } from '@ui/context/colors';
import { ColorMapHex, parseColorAnnotation } from '@ui/features/stage/utils';
import { useGetPromotion } from '@ui/gen/api/v2/core/core';
import { PromotionReference, PromotionRequestReference, Stage } from '@ui/gen/api/v2/models';
import { getContrastTextColor } from '@ui/utils/get-contrast-text-color';

import { IAction, useActionContext } from '../context/action-context';
import { useDictionaryContext } from '../context/dictionary-context';
import { useFreightTimelineControllerContext } from '../context/freight-timeline-controller-context';

export const isStageControlFlow = (stage: Stage) =>
  (stage?.spec?.promotionTemplate?.spec?.steps?.length || 0) <= 0;

// isStageTargetAware reports whether a Stage governs Targets, and therefore
// promotes Freight via a PromotionRequest rather than a Promotion.
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
// promotes through a PromotionRequest, which fans out one child Promotion per
// Target, and records the request in status.currentPromotionRequest and
// status.lastPromotionRequest instead. For such a Stage the Promotion fields
// name at most one of the children, so they never stand for the round.
export type StagePromotionRef = {
  kind: 'Promotion' | 'PromotionRequest';
  name?: string;
  phase?: string;
  message?: string;
  finishedAt?: string;
  freightName?: string;
  // path is where the UI shows the referenced promotion activity. A Promotion
  // has a page of its own. A PromotionRequest does not: it is listed on the
  // Stage page, whose Promotions tab leads with the Stage's PromotionRequests
  // and expands the one currently fanning out.
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
    path: stagePath(stage)
  };

// getCurrentPromotionRef describes what the Stage is promoting right now, if
// anything.
export const getCurrentPromotionRef = (stage?: Stage): StagePromotionRef | undefined =>
  isStageTargetAware(stage)
    ? promotionRequestRef(stage, stage?.status?.currentPromotionRequest)
    : promotionRef(stage, stage?.status?.currentPromotion);

// getLastPromotionRef describes the last promotion the Stage saw through to a
// terminal phase, if any.
export const getLastPromotionRef = (stage?: Stage): StagePromotionRef | undefined =>
  isStageTargetAware(stage)
    ? promotionRequestRef(stage, stage?.status?.lastPromotionRequest)
    : promotionRef(stage, stage?.status?.lastPromotion);

export const getLastPromotionDate = (stage: Stage) => {
  const finishedAt = getLastPromotionRef(stage)?.finishedAt;
  return finishedAt ? new Date(finishedAt) : null;
};

// getCurrentPromotion names the Promotion a Stage is currently promoting
// through, for callers that need to read the Promotion itself. A target-aware
// Stage promotes through a PromotionRequest, whose child Promotions target
// distinct Targets and run in parallel, so no single Promotion describes what
// it is doing: for such a Stage this is always undefined.
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

export const useStageHeaderStyle = (stage: Stage): CSSProperties => {
  const colorContext = useContext(ColorContext);
  if (!useIsColorsUsed()) {
    return {};
  }

  let stageColor =
    parseColorAnnotation(stage) || colorContext.stageColorMap[stage?.metadata?.name || ''];
  let stageFontColor = '';

  if (stageColor && ColorMapHex[stageColor]) {
    stageColor = ColorMapHex[stageColor];
  }

  if (stageColor) {
    stageFontColor = getContrastTextColor(ColorMapHex[stageColor] || stageColor);
  }

  return {
    backgroundColor: stageColor || '',
    color: stageFontColor
  };
};
