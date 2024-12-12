// 1. Use case: Deep links
// Scopes:
// - PromotionStep:
//      Plugin generates information from given plugin step and renders in the step view in Promotion steps modal
// - Promotion:
//      Plugin summarises promotion and provide useful link(s) and renders in the promotion list view
// - Stage:
//      Plugin summarises stage and provide useful link(s) and renders at the top in stage details page
// - Pipeline Stage:
//      Similar to stage but since this is in Pipeline and directly accessible so needs carefully show

import { ReactNode } from 'react';

import { PromotionDirectiveStepStatus } from '@ui/features/common/promotion-directive-step-status/utils';
import { PromotionStep } from '@ui/gen/v1alpha1/generated_pb';

export interface DeepLinkPluginsInstallation {
  // Scopes

  // Plugin generates information from given plugin step and renders in the step view in Promotion steps modal
  PromotionStep: {
    // thoughts.. instead of coupling this to name of step, this would open doors for plugin development based on combination of promotion steps
    shouldRender: (opts: { step: PromotionStep; result: PromotionDirectiveStepStatus }) => boolean;
    render: (props: {
      // step metadata
      step: PromotionStep;
      // flexible to render based on status
      result: PromotionDirectiveStepStatus;
      // output from steps
      output?: Record<string, unknown>;
    }) => ReactNode;
  };
}

export interface PluginsInstallation {
  identity?: string;
  description?: string;

  DeepLinkPlugin: DeepLinkPluginsInstallation;
}
