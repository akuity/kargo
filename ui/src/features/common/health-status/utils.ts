import { HealthState } from '@ui/gen/v1alpha1/types_pb';

export const healthStateToString = (status?: HealthState): string => {
  switch (status) {
    case HealthState.HEALTHY:
      return 'Healthy';
    case HealthState.UNHEALTHY:
      return 'Unhealthy';
    case HealthState.UNKNOWN:
      return 'Unknown';
    default:
      return '';
  }
};
