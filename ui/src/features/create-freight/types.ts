import {
  ChartDiscoveryResult,
  DiscoveredCommit,
  DiscoveredImageReference,
  GitDiscoveryResult,
  ImageDiscoveryResult
} from '@ui/gen/v1alpha1/generated_pb';

export type DiscoveryResult = ImageDiscoveryResult | ChartDiscoveryResult | GitDiscoveryResult;
export type FreightInfo = DiscoveredImageReference | string | DiscoveredCommit;
