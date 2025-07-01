import { faExternalLink } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tag } from 'antd';
import Link from 'antd/es/typography/Link';
import { ReactNode } from 'react';

import {
  getGitCommitURL,
  getImageSource
} from '@ui/features/freight-timeline/open-container-initiative-utils';
import { Chart, GitCommit, Image } from '@ui/gen/api/v1alpha1/generated_pb';

import { humanComprehendableArtifact } from './artifact-parts-utils';
import { shortVersion } from './short-version-utils';

type FreightArtifactProps = {
  artifact: GitCommit | Chart | Image;
  expand?: boolean;
};

export const FreightArtifact = (props: FreightArtifactProps) => {
  const artifactType = props.artifact?.$typeName;

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

    const TagComponent = (
      <Tag title={props.artifact.repoURL} bordered={false} color='geekblue' key={props.artifact.id}>
        {props.artifact.id.slice(0, 7)}

        {!!url && (
          <FontAwesomeIcon icon={faExternalLink} className='text-blue-600 text-[8px] ml-1' />
        )}

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
    >
      {shortVersion(props.artifact?.tag)}

      {!!imageSourceFromOci && (
        <FontAwesomeIcon icon={faExternalLink} className='text-blue-600 ml-1 text-[8px]' />
      )}

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
