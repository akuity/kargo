import {
  TableSource,
  flattenFreightOrigin
} from '@ui/features/freight/flatten-freight-origin-utils';
import { FreightCollection } from '@ui/gen/api/v2/models';

export type Param = { key: string; value: string };

// flattenParams reads a Target's params into dotted keys, the way a promotion
// step's expression reaches them (`target.params.cluster.region`). Arrays and
// other values print as JSON, since a step reads those whole.
export const flattenParams = (params: Record<string, unknown> = {}): Param[] => {
  const walk = (value: Record<string, unknown>, prefix: string): Param[] =>
    Object.entries(value).flatMap(([key, nested]) => {
      const path = prefix ? `${prefix}.${key}` : key;
      if (nested && typeof nested === 'object' && !Array.isArray(nested)) {
        const inner = walk(nested as Record<string, unknown>, path);
        // An object with nothing in it still has something to say, and
        // flattening it away would take the key off the page entirely.
        return inner.length ? inner : [{ key: path, value: '{}' }];
      }
      return [
        {
          key: path,
          value: typeof nested === 'string' ? nested : (JSON.stringify(nested) ?? '')
        }
      ];
    });
  return walk(params, '').sort((lhs, rhs) => lhs.key.localeCompare(rhs.key));
};

// repoName is the last segment of a repository URL -- `otel-app` out of
// `ghcr.io/example/otel-app` -- which is what names an artifact in a line too
// short for the whole URL.
const repoName = (repoURL: string): string =>
  repoURL.replace(/\/+$/, '').split('/').pop() || repoURL;

export type Artifact = { kind: string; text: string };

// describeArtifact names one artifact of a piece of Freight in a few
// characters: an image by tag, a chart by version, a commit by short SHA.
export const describeArtifact = (source: TableSource): Artifact => {
  switch (source.type) {
    case 'image':
      return { kind: 'Image', text: `${repoName(source.repoURL)}:${source.tag || 'latest'}` };
    case 'helm':
      return {
        kind: 'Chart',
        text: `${source.name || repoName(source.repoURL)}:${source.version}`
      };
    case 'git':
      return { kind: 'Commit', text: `${repoName(source.repoURL)}@${source.id.slice(0, 7)}` };
    default:
      return {
        kind: source.artifactType || 'Artifact',
        text: source.version || source.subscriptionName || ''
      };
  }
};

// freightArtifacts describes everything a Freight collection carries, across
// every origin in it.
export const freightArtifacts = (collection?: FreightCollection): Artifact[] =>
  Object.values(collection?.items || {})
    .flatMap((reference) => flattenFreightOrigin(reference))
    .map(describeArtifact)
    .filter(({ text }) => text);
