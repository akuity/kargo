import { Runner } from '@ui/features/promotion-directives/registry/types';

export type RunnerWithConfiguration /* configuration that users do in wizard */ = Runner & {
  state?: object; // object that we don't care about. this mostly handled by rsjf
  as?: string; // added this so that state does not loose actual value - not a concern now but when we have "edit" stage wizard
};
