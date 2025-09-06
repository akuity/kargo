import { faMagicWandSparkles } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import Alert from 'antd/es/alert/Alert';
import classNames from 'classnames';
import { ReactNode } from 'react';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { Chart, Freight, GitCommit, Image } from '@ui/gen/api/v1alpha1/generated_pb';

export const SimilarToFreightNote = (props: {
  similarToFreight?: Freight;
  missingArtifacts: (Image | GitCommit | Chart)[];
  className?: string;
}) => {
  if (!props.similarToFreight) {
    return null;
  }

  const allMissing =
    props.missingArtifacts.length ===
    props.similarToFreight?.images?.length +
      props.similarToFreight?.charts?.length +
      props.similarToFreight?.commits?.length;

  if (allMissing) {
    return null;
  }

  let description: ReactNode;

  if (props.missingArtifacts.length > 0) {
    description = (
      <>
        {props.missingArtifacts.map((artifact, idx) => {
          const isLast = props.missingArtifacts?.length === idx + 1;

          if (artifact.$typeName === 'github.com.akuity.kargo.api.v1alpha1.Image') {
            return (
              <>
                {artifact?.repoURL}:{artifact.tag}
                {!isLast && <>, </>}
              </>
            );
          }

          if (artifact.$typeName === 'github.com.akuity.kargo.api.v1alpha1.Chart') {
            return (
              <>
                {artifact?.repoURL}:{artifact.version}
                {!isLast && <>, </>}
              </>
            );
          }

          return (
            <>
              {artifact.repoURL}
              {artifact.tag ? `:${artifact.tag}` : `/${artifact.id}`}
              {!isLast && <>, </>}
            </>
          );
        })}{' '}
        not available, defaulted to latest discovered version.
      </>
    );
  }

  return (
    <Alert
      type='info'
      className={classNames(props.className)}
      message={
        <>
          Based on{' '}
          <Link
            to={generatePath(paths.freight, {
              name: props.similarToFreight?.metadata?.namespace,
              freightName: props.similarToFreight?.metadata?.name
            })}
          >
            {props.similarToFreight?.alias}
          </Link>{' '}
          - matching versions are pre-filled.
        </>
      }
      icon={<FontAwesomeIcon icon={faMagicWandSparkles} />}
      showIcon
      description={description}
    />
  );
};
