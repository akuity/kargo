import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { faAnchor, IconDefinition } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import classNames from 'classnames';

import { Chart, GitCommit, Image } from '@ui/gen/api/v1alpha1/generated_pb';

export const ArtifactIcon = (props: {
  artifactType: GitCommit['$typeName'] | Chart['$typeName'] | Image['$typeName'];
  className?: string;
}) => {
  let icon: IconDefinition;

  switch (props.artifactType) {
    case 'github.com.akuity.kargo.api.v1alpha1.GitCommit':
      icon = faGit;
      break;

    case 'github.com.akuity.kargo.api.v1alpha1.Chart':
      icon = faAnchor;
      break;

    case 'github.com.akuity.kargo.api.v1alpha1.Image':
      icon = faDocker;
      break;
  }

  return <FontAwesomeIcon icon={icon} className={classNames(props.className, 'text-[11px]')} />;
};
