import { CSSProperties, useContext, useMemo } from 'react';

import { ColorContext } from '@ui/context/colors';
import { ColorMapHex, parseColorAnnotation } from '@ui/features/stage/utils';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { IAction, useActionContext } from '../context/action-context';
import { useDictionaryContext } from '../context/dictionary-context';
import { useFreightTimelineControllerContext } from '../context/freight-timeline-controller-context';

export const isStageControlFlow = (stage: Stage) =>
  (stage?.spec?.promotionTemplate?.spec?.steps?.length || 0) <= 0;

export const getStageHealth = (stage: Stage) => stage?.status?.health;

export const useIsColorsUsed = () => {
  const freightTimelineControllerContext = useFreightTimelineControllerContext();

  return freightTimelineControllerContext?.preferredFilter?.showColors;
};

export const getLastPromotionDate = (stage: Stage) => stage?.status?.lastPromotion?.finishedAt;

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
    stageFontColor = 'white';
  }

  if (stageColor) {
    stageFontColor = 'white';
  }

  return {
    backgroundColor: stageColor || '',
    color: stageFontColor
  };
};
