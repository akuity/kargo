import { Tag } from 'antd';
import Link from 'antd/es/typography/Link';
import { ReactNode } from 'react';

import {
  getGitCommitURL,
  getImageSource
} from '@ui/features/freight-timeline/open-container-initiative-utils';
import {
  Chart,
  ArtifactReference as GenericArtifactReference,
  GitCommit,
  Image
} from '@ui/gen/api/v1alpha1/generated_pb';

import { ArtifactIcon } from './artifact-icon';
import { humanComprehendableArtifact } from './artifact-parts-utils';
import { shortVersion } from './short-version-utils';

type FreightArtifactProps = {
  artifact: GitCommit | Chart | Image | GenericArtifactReference;
  expand?: boolean;
};

export const FreightArtifact = (props: FreightArtifactProps) => {
  const artifactType = props.artifact?.$typeName;

  if (artifactType === 'github.com.akuity.kargo.api.v1alpha1.ArtifactReference') {
    return (
      <Tag color='geekblue' bordered={false}>
        {shortVersion(props.artifact.version)}
      </Tag>
    );
  }

  let Expand: ReactNode;

  if (props.expand) {
    Expand = (
      <span className='text-[10px] ml-1'>
        {humanComprehendableArtifact(props.artifact.repoURL)}
      </span>
    );
  }

  if (artifactType === 'github.com.akuity.kargo.api.v1alpha1.GitCommit') {
    const url = getGitCommitURL(props.artifact.repoURL, props.artifact.id);

    // prioritize semver
    const id = props.artifact.tag || props.artifact.id;

    const TagComponent = (
      <Tag title={props.artifact.repoURL} bordered={false} color='geekblue' key={props.artifact.id}>
        <ArtifactIcon artifactType={artifactType} className='mr-1' />

        {id.slice(0, 7)}

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

  if (artifactType === 'github.com.akuity.kargo.api.v1alpha1.Chart') {
    return (
      <Tag
        title={`${props.artifact.repoURL}:${props.artifact.version}`}
        bordered={false}
        color='geekblue'
        key={props.artifact.repoURL}
      >
        <ArtifactIcon artifactType={artifactType} className='mr-1' />

        {shortVersion(props.artifact.version)}

        {Expand}
      </Tag>
    );
  }

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
      <ArtifactIcon artifactType={artifactType} className='mr-1' />

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
};
