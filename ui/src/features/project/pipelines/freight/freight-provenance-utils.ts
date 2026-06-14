import {
  ArtifactReference as GenericArtifactReference,
  Chart,
  Freight,
  GitCommit,
  Image
} from '@ui/gen/api/v1alpha1/generated_pb';

export type FreightProvenanceRow = {
  key: string;
  label: string;
  value: string;
  title?: string;
};

const shortID = (id: string = '', length = 7) => id.slice(0, length);

export const artifactName = (repoURL: string = '') => {
  const normalized = repoURL.replace(/\/+$/, '').replace(/\.git$/, '');
  const parts = normalized.split('/').filter(Boolean);
  return parts[parts.length - 1] || repoURL;
};

const gitRow = (commit: GitCommit, index: number): FreightProvenanceRow | null => {
  const revision = commit.tag || shortID(commit.id);
  if (!revision) {
    return null;
  }
  const source = artifactName(commit.repoURL);
  return {
    key: `git-${commit.repoURL || index}-${revision}`,
    label: 'git',
    value: source ? `${source} @ ${revision}` : revision,
    title: `${commit.repoURL || 'Git'}${commit.id ? ` @ ${commit.id}` : ''}`
  };
};

const chartRow = (chart: Chart, index: number): FreightProvenanceRow | null => {
  if (!chart.version) {
    return null;
  }
  const source = chart.name || artifactName(chart.repoURL);
  return {
    key: `chart-${chart.repoURL || index}-${chart.name || ''}-${chart.version}`,
    label: 'chart',
    value: source ? `${source}:${chart.version}` : chart.version,
    title: `${chart.repoURL || 'Chart'}${chart.name ? `/${chart.name}` : ''}:${chart.version}`
  };
};

const imageRow = (image: Image, index: number): FreightProvenanceRow | null => {
  if (!image.tag) {
    return null;
  }
  const source = artifactName(image.repoURL);
  return {
    key: `image-${image.repoURL || index}-${image.tag}`,
    label: 'image',
    value: source ? `${source}:${image.tag}` : image.tag,
    title: `${image.repoURL || 'Image'}:${image.tag}`
  };
};

const genericArtifactRow = (
  artifact: GenericArtifactReference,
  index: number
): FreightProvenanceRow | null => {
  if (!artifact.version) {
    return null;
  }
  return {
    key: `artifact-${artifact.subscriptionName || index}-${artifact.version}`,
    label: 'artifact',
    value: artifact.subscriptionName
      ? `${artifact.subscriptionName}:${artifact.version}`
      : artifact.version,
    title: `${artifact.artifactType || 'Artifact'}:${artifact.version}`
  };
};

export const freightShortName = (freight: Freight) => shortID(freight.metadata?.name || '');

export const getFreightProvenanceRows = (
  freight: Freight,
  options: { showAlias: boolean; age?: string } = { showAlias: true }
): FreightProvenanceRow[] => {
  const rows: FreightProvenanceRow[] = [];

  if (options.showAlias && freight.alias) {
    rows.push({
      key: 'alias',
      label: 'alias',
      value: freight.alias,
      title: freight.alias
    });
  }

  rows.push(
    ...freight.commits
      .map((commit, index) => gitRow(commit, index))
      .filter((row): row is FreightProvenanceRow => Boolean(row)),
    ...freight.charts
      .map((chart, index) => chartRow(chart, index))
      .filter((row): row is FreightProvenanceRow => Boolean(row)),
    ...freight.images
      .map((image, index) => imageRow(image, index))
      .filter((row): row is FreightProvenanceRow => Boolean(row)),
    ...freight.artifacts
      .map((artifact, index) => genericArtifactRow(artifact, index))
      .filter((row): row is FreightProvenanceRow => Boolean(row))
  );

  const id = freightShortName(freight);
  if (id) {
    rows.push({
      key: 'id',
      label: 'id',
      value: id,
      title: freight.metadata?.name
    });
  }

  if (options.age) {
    rows.push({
      key: 'age',
      label: 'age',
      value: options.age
    });
  }

  return rows;
};
