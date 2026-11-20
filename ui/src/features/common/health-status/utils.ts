import { Tone } from '@ui/features/common/status-tone';

export enum HealthStatus {
  HEALTHY = 'Healthy',
  DEGRADED = 'Degraded',
  PROGRESSING = 'Progressing',
  UNHEALTHY = 'Unhealthy',
  UNKNOWN = 'Unknown',
  UNDEFINED = ''
}

export const healthStatusToEnum = (status?: string): HealthStatus => {
  switch (status) {
    case HealthStatus.HEALTHY:
      return HealthStatus.HEALTHY;
    case HealthStatus.PROGRESSING:
      return HealthStatus.PROGRESSING;
    case HealthStatus.DEGRADED:
      return HealthStatus.DEGRADED;
    case HealthStatus.UNHEALTHY:
      return HealthStatus.UNHEALTHY;
    case HealthStatus.UNKNOWN:
      return HealthStatus.UNKNOWN;
    default:
      return HealthStatus.UNDEFINED;
  }
};

// How a health status reads. Degraded and Unhealthy are both failures -- the
// difference between them is in the word, not the color -- and a status the
// API reports that we do not know steps back rather than guessing at one.
export const healthTone = (status?: string): Tone => {
  switch (healthStatusToEnum(status)) {
    case HealthStatus.HEALTHY:
      return 'success';
    case HealthStatus.DEGRADED:
    case HealthStatus.UNHEALTHY:
      return 'error';
    case HealthStatus.PROGRESSING:
      return 'processing';
    case HealthStatus.UNKNOWN:
      return 'warning';
    default:
      return 'muted';
  }
};
