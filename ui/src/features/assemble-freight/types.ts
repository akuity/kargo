import {
  ChartDiscoveryResult,
  GitDiscoveryResult,
  ImageDiscoveryResult,
  DiscoveryResult as GenericDiscoveryResult,
  ArtifactReference,
  DiscoveredCommit,
  DiscoveredImageReference
} from '@ui/gen/api/v2/models';

export type DiscoveryResult =
  ImageDiscoveryResult | ChartDiscoveryResult | GitDiscoveryResult | GenericDiscoveryResult;

export type FreightInfo = DiscoveredImageReference | string | DiscoveredCommit | ArtifactReference;
