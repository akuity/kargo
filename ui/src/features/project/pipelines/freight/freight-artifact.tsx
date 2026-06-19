import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { faAnchor } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tag } from 'antd';
import Link from 'antd/es/typography/Link';

import {
  getGitCommitURL,
  getImageSource
} from '@ui/features/freight-timeline/open-container-initiative-utils';
import { ArtifactReference, Chart, GitCommit, Image } from '@ui/gen/api/v2/models';

import { humanComprehendableArtifact } from './artifact-parts-utils';
import { shortVersion } from './short-version-utils';

const ArtifactName = (props: { artifact: GitCommit | Chart | Image }) => (
  <span className='text-[10px] ml-1'>{humanComprehendableArtifact(props.artifact)}</span>
);

export const GitCommitArtifact = (props: { commit: GitCommit; expand?: boolean }) => {
  const { commit } = props;

  const url = getGitCommitURL(commit.repoURL || '', commit.id || '');

  // prioritize semver; use shortVersion for tags, 7-char slice for raw commit hashes
  const displayId = commit.tag ? shortVersion(commit.tag) : commit.id?.slice(0, 7);

  const TagComponent = (
    <Tag title={commit.repoURL} bordered={false} color='geekblue' key={commit.id}>
      <FontAwesomeIcon icon={faGit} className='mr-1 text-[11px]' />

      {displayId}

      {props.expand && <ArtifactName artifact={commit} />}
    </Tag>
  );

  if (url) {
    return (
      <Link key={commit.repoURL} href={url} target='_blank' onClick={(e) => e.stopPropagation()}>
        {TagComponent}
      </Link>
    );
  }

  return TagComponent;
};

export const ChartArtifact = (props: { chart: Chart; expand?: boolean }) => {
  const { chart } = props;

  return (
    <Tag
      title={`${chart.repoURL}:${chart.version}`}
      bordered={false}
      color='geekblue'
      key={chart.repoURL}
    >
      <FontAwesomeIcon icon={faAnchor} className='mr-1 text-[11px]' />

      {shortVersion(chart.version)}

      {props.expand && <ArtifactName artifact={chart} />}
    </Tag>
  );
};

export const ImageArtifact = (props: { image: Image; expand?: boolean }) => {
  const { image } = props;

  let imageSourceFromOci = '';

  if (image.annotations) {
    imageSourceFromOci = getImageSource(image.annotations);
  }

  const TagComponent = (
    <Tag
      title={`${image.repoURL}:${image.tag}`}
      bordered={false}
      color='geekblue'
      key={image?.repoURL}
      className='hover:cursor-default'
    >
      <FontAwesomeIcon icon={faDocker} className='mr-1 text-[11px]' />

      {shortVersion(image?.tag)}

      {props.expand && <ArtifactName artifact={image} />}
    </Tag>
  );

  if (imageSourceFromOci) {
    return (
      <Link
        key={image?.repoURL}
        href={imageSourceFromOci}
        target='_blank'
        onClick={(e) => e.stopPropagation()}
      >
        {TagComponent}
      </Link>
    );
  }

  return TagComponent;
};

export const GenericArtifact = (props: { artifact: ArtifactReference }) => (
  <Tag color='geekblue' bordered={false}>
    {shortVersion(props.artifact.version)}
  </Tag>
);
