import { faMagicWandSparkles } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import Alert from 'antd/es/alert/Alert';
import classNames from 'classnames';
import { ReactNode } from 'react';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { Chart, Freight, GitCommit, Image } from '@ui/gen/api/v1alpha1/generated_pb';

export const CloneFreightNote = (props: {
  cloneFreight?: Freight;
  missingArtifacts: (Image | GitCommit | Chart)[];
  className?: string;
}) => {
  if (!props.cloneFreight) {
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
              name: props.cloneFreight?.metadata?.namespace,
              freightName: props.cloneFreight?.metadata?.name
            })}
          >
            {props.cloneFreight?.alias}
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
