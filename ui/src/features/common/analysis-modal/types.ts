import {
  Measurement,
  Metric,
  MetricResult
} from '@ui/gen/api/stubs/rollouts/v1alpha1/generated_pb';

export enum AnalysisStatus {
  Successful = 'Successful',
  Error = 'Error',
  Failed = 'Failed',
  Running = 'Running',
  Pending = 'Pending',
  Inconclusive = 'Inconclusive',
  Unknown = 'Unknown' // added by frontend
}

export enum FunctionalStatus {
  ERROR = 'ERROR',
  INACTIVE = 'INACTIVE',
  IN_PROGRESS = 'IN_PROGRESS',
  SUCCESS = 'SUCCESS',
  WARNING = 'WARNING'
}

export type TransformedMetricStatus = MetricResult & {
  adjustedPhase: AnalysisStatus;
  chartable: boolean;
  chartMax?: number;
  chartMin: number;
  statusLabel: string;
  substatus?: FunctionalStatus.ERROR | FunctionalStatus.WARNING;
  transformedMeasurements: TransformedMeasurement[];
};

export type TransformedMetricSpec = Metric & {
  failConditionLabel?: string;
  failThresholds?: number[];
  queries?: string[];
  successConditionLabel?: string;
  successThresholds?: number[];
  conditionKeys: string[];
};

export type TransformedMetric = {
  name: string;
  spec?: TransformedMetricSpec;
  status: TransformedMetricStatus;
};

export type TransformedValueObject = {
  [key: string]: number | string | null;
};

export type TransformedMeasurement = Measurement & {
  chartValue?: TransformedValueObject | number | string | null;
  tableValue: TransformedValueObject | number | string | null;
};

export type MeasurementSetInfo = {
  chartable: boolean;
  max: number | null;
  measurements: TransformedMeasurement[];
  min: number;
};

export type MeasurementValueInfo = {
  canChart: boolean;
  chartValue?: TransformedValueObject | number | string | null;
  tableValue: TransformedValueObject | number | string | null;
};
