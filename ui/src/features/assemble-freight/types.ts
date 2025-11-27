import {
  ChartDiscoveryResult,
  DiscoveredCommit,
  DiscoveredImageReference,
  GenericArtifactReference,
  GenericDiscoveryResult,
  GitDiscoveryResult,
  ImageDiscoveryResult
} from '@ui/gen/api/v1alpha1/generated_pb';

export type DiscoveryResult =
  | ImageDiscoveryResult
  | ChartDiscoveryResult
  | GitDiscoveryResult
  | GenericDiscoveryResult;
export type FreightInfo =
  | DiscoveredImageReference
  | string
  | DiscoveredCommit
  | GenericArtifactReference;
