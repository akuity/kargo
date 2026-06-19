import { faMagicWandSparkles } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import Alert from 'antd/es/alert/Alert';
import classNames from 'classnames';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { Freight } from '@ui/gen/api/v2/models';

import { MissingArtifacts } from './missing-artifacts-to-cloned-freight';

export const CloneFreightNote = (props: {
  cloneFreight?: Freight;
  missingArtifacts: MissingArtifacts;
  className?: string;
}) => {
  if (!props.cloneFreight) {
    return null;
  }

  const { images, charts, commits } = props.missingArtifacts;

  // Each artifact type keeps its own array, so the type is known here without
  // inspecting any artifact's shape; we format each list and join them together.
  const missing = [
    ...images.map((image) => `${image.repoURL}:${image.tag}`),
    ...charts.map((chart) => `${chart.repoURL}:${chart.version}`),
    ...commits.map(
      (commit) => `${commit.repoURL}${commit.tag ? `:${commit.tag}` : `/${commit.id}`}`
    )
  ];

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
      description={
        missing.length > 0 ? (
          <>{missing.join(', ')} not available, defaulted to latest discovered version.</>
        ) : undefined
      }
    />
  );
};
