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
  | ImageDiscoveryResult
  | ChartDiscoveryResult
  | GitDiscoveryResult
  | GenericDiscoveryResult;

export type FreightInfo = DiscoveredImageReference | string | DiscoveredCommit | ArtifactReference;

// The active selection in the assembly sidebar. Each menu group renders a single
// artifact kind, so the kind is known when an item is selected and is carried
// here -- there is no need to sniff the shape of `value` afterwards.
export type SelectedArtifact =
  | { kind: 'image'; value: ImageDiscoveryResult }
  | { kind: 'chart'; value: ChartDiscoveryResult }
  | { kind: 'git'; value: GitDiscoveryResult }
  | { kind: 'generic'; value: GenericDiscoveryResult };

// The artifacts a user has chosen to include in the new Freight, kept in
// separate per-kind buckets so each can carry its concrete artifact and info
// types without a union.
export type ChosenItems = {
  image: Record<string, { artifact: ImageDiscoveryResult; info: DiscoveredImageReference }>;
  chart: Record<string, { artifact: ChartDiscoveryResult; info: string }>;
  git: Record<string, { artifact: GitDiscoveryResult; info: DiscoveredCommit }>;
  generic: Record<string, { artifact: GenericDiscoveryResult; info: ArtifactReference }>;
};
