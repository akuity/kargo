import { useState } from 'react';

import { FreightMode, FreightlineAction } from '../types';

export interface PipelineStateHook {
  action?: FreightlineAction;
  freight?: string;
  stage?: string;
  clear: () => void;
  select: (action?: FreightlineAction, stage?: string, freight?: string) => void;
}

export const usePipelineState = (): PipelineStateHook => {
  const [action, setAction] = useState<FreightlineAction | undefined>();
  const [freight, setFreight] = useState<string | undefined>();
  const [stage, setStage] = useState<string | undefined>();

  const clear = () => {
    setAction(undefined);
    setFreight(undefined);
    setStage(undefined);
  };

  const select = (_action?: FreightlineAction, _stage?: string, _freight?: string) => {
    if (action === _action) {
      clear();
      return;
    }
    if (_action) {
      setAction(_action);
      if (
        _action === FreightlineAction.Promote ||
        _action === FreightlineAction.PromoteSubscribers
      ) {
        setStage(_stage);
        setFreight(undefined);
      } else if (_action === FreightlineAction.ManualApproval) {
        setFreight(_freight);
        setStage(undefined);
      }
    } else {
      if (_stage) {
        setStage(_stage);
      }
      if (_freight) {
        setFreight(_freight === freight ? undefined : _freight); // deselect if already selected
      }
    }
  };

  return {
    action,
    freight,
    stage,
    clear,
    select
  };
};

export const isPromoting = ({ action, stage }: PipelineStateHook) => {
  return (
    stage &&
    (action === FreightlineAction.PromoteSubscribers || action === FreightlineAction.Promote)
  );
};

export const getFreightMode = (
  state: PipelineStateHook,
  freightID: string,
  promotionEligible: boolean
): FreightMode => {
  if (state.action === FreightlineAction.ManualApproval) {
    return state.freight === freightID ? FreightMode.Selected : FreightMode.Disabled;
  }

  if (!state.stage) {
    // not promoting or confirming
    return FreightMode.Default;
  }

  if (state.freight === freightID) {
    return FreightMode.Confirming;
  }

  return promotionEligible ? FreightMode.Promotable : FreightMode.Disabled;
};
