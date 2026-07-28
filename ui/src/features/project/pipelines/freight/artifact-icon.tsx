import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { faAnchor, faBox, IconDefinition } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import classNames from 'classnames';

import {
  isArtifactChart,
  isArtifactGitCommit,
  isArtifactImage
} from '@ui/features/assemble-freight/artifact-type-guards';
import { ArtifactReference, Chart, GitCommit, Image } from '@ui/gen/api/v2/models';

export const ArtifactIcon = (props: {
  artifact: GitCommit | Chart | Image | ArtifactReference;
  className?: string;
}) => {
  let icon: IconDefinition = faBox;

  if (isArtifactImage(props.artifact)) {
    icon = faDocker;
  } else if (isArtifactGitCommit(props.artifact)) {
    icon = faGit;
  } else if (isArtifactChart(props.artifact)) {
    icon = faAnchor;
  }

  return <FontAwesomeIcon icon={icon} className={classNames(props.className, 'text-[11px]')} />;
};
