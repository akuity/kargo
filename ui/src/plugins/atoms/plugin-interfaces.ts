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
import { Promotion, PromotionStep } from '@ui/gen/v1alpha1/generated_pb';

interface DeepLinksPluginProps {
  PromotionStep: {
    // step metadata
    step: PromotionStep;
    // flexible to render based on status
    result: PromotionDirectiveStepStatus;
    // output from steps
    output?: Record<string, unknown>;
  };

  Promotion: {
    promotion: Promotion;

    // which argocd shard does this promotion affect
    unstable_argocdShardUrl?: string;

    isLatestPromotion?: boolean;
  };
}

export interface DeepLinkPluginsInstallation {
  // Scopes

  // Plugin generates information from given plugin step and renders in the step view in Promotion steps modal
  PromotionStep?: {
    // thoughts.. instead of coupling this to name of step, this would open doors for plugin development based on combination of promotion steps
    shouldRender: (opts: DeepLinksPluginProps['PromotionStep']) => boolean;
    render: (props: DeepLinksPluginProps['PromotionStep']) => ReactNode;
  };

  // Plugin summarises promotion and provide useful link(s) and renders in the promotion list view
  Promotion?: {
    shouldRender: (opts: DeepLinksPluginProps['Promotion']) => boolean;
    render: (props: DeepLinksPluginProps['Promotion']) => ReactNode;
  };
}

export interface PluginsInstallation {
  identity?: string;
  description?: string;

  DeepLinkPlugin?: DeepLinkPluginsInstallation;
}
