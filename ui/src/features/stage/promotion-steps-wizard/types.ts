import { Runner } from '@ui/features/promotion-directives/registry/types';
import { PromotionStepConfig } from '@ui/gen/api/v2/models';

export type RunnerWithConfiguration /* configuration that users do in wizard */ = Runner & {
  state?: PromotionStepConfig; // object that we don't care about. this mostly handled by rsjf
  as?: string; // added this so that state does not loose actual value - not a concern now but when we have "edit" stage wizard
  continueOnError?: boolean;
};
