export enum HealthStatus {
  HEALTHY = 'Healthy',
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
    case HealthStatus.UNHEALTHY:
      return HealthStatus.UNHEALTHY;
    case HealthStatus.UNKNOWN:
      return HealthStatus.UNKNOWN;
    default:
      return HealthStatus.UNDEFINED;
  }
};
