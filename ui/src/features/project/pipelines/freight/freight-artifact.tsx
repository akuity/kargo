import { Tag } from 'antd';
import Link from 'antd/es/typography/Link';
import { ReactNode } from 'react';

import {
  isArtifactChart,
  isArtifactGitCommit,
  isArtifactImage
} from '@ui/features/assemble-freight/artifact-type-guards';
import {
  getGitCommitURL,
  getImageSource
} from '@ui/features/freight-timeline/open-container-initiative-utils';
import { ArtifactReference, Chart, GitCommit, Image } from '@ui/gen/api/v2/models';

import { ArtifactIcon } from './artifact-icon';
import { humanComprehendableArtifact } from './artifact-parts-utils';
import { shortVersion } from './short-version-utils';

type FreightArtifactProps = {
  artifact: GitCommit | Chart | Image | ArtifactReference;
  expand?: boolean;
};

export const FreightArtifact = (props: FreightArtifactProps) => {
  let Expand: ReactNode;

  if (props.expand) {
    Expand = (
      <span className='text-[10px] ml-1'>{humanComprehendableArtifact(props.artifact)}</span>
    );
  }

  if (isArtifactGitCommit(props.artifact)) {
    const url = getGitCommitURL(props.artifact.repoURL || '', props.artifact.id || '');

    // prioritize semver; use shortVersion for tags, 7-char slice for raw commit hashes
    const displayId = props.artifact.tag
      ? shortVersion(props.artifact.tag)
      : props.artifact.id?.slice(0, 7);

    const TagComponent = (
      <Tag title={props.artifact.repoURL} bordered={false} color='geekblue' key={props.artifact.id}>
        <ArtifactIcon artifact={props.artifact} className='mr-1' />

        {displayId}

        {Expand}
      </Tag>
    );

    if (url) {
      return (
        <Link
          key={props.artifact.repoURL}
          href={url}
          target='_blank'
          onClick={(e) => e.stopPropagation()}
        >
          {TagComponent}
        </Link>
      );
    }

    return TagComponent;
  }

  if (isArtifactChart(props.artifact)) {
    return (
      <Tag
        title={`${props.artifact.repoURL}:${props.artifact.version}`}
        bordered={false}
        color='geekblue'
        key={props.artifact.repoURL}
      >
        <ArtifactIcon artifact={props.artifact} className='mr-1' />

        {shortVersion(props.artifact.version)}

        {Expand}
      </Tag>
    );
  }

  if (isArtifactImage(props.artifact)) {
    let imageSourceFromOci = '';

    if (props.artifact.annotations) {
      imageSourceFromOci = getImageSource(props.artifact.annotations);
    }

    const TagComponent = (
      <Tag
        title={`${props.artifact.repoURL}:${props.artifact.tag}`}
        bordered={false}
        color='geekblue'
        key={props.artifact?.repoURL}
        className='hover:cursor-default'
      >
        <ArtifactIcon artifact={props.artifact} className='mr-1' />

        {shortVersion(props.artifact?.tag)}

        {Expand}
      </Tag>
    );

    if (imageSourceFromOci) {
      return (
        <Link
          key={props.artifact?.repoURL}
          href={imageSourceFromOci}
          target='_blank'
          onClick={(e) => e.stopPropagation()}
        >
          {TagComponent}
        </Link>
      );
    }

    return TagComponent;
  }

  return (
    <Tag color='geekblue' bordered={false}>
      {shortVersion(props.artifact.version)}
    </Tag>
  );
};
