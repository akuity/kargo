import { faChevronLeft, faChevronRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button } from 'antd';

import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

import { useFreightTimelineControllerContext } from '../context/freight-timeline-controller-context';

import {
  selectActiveCarouselFreight,
  selectNextArtifact,
  selectPreviousArtifact
} from './artifact-selector-utils';
import { FreightArtifact } from './freight-artifact';

type FreightArtifactCarouselProps = {
  freight: Freight;
};

export const FreightArtifactCarousel = (props: FreightArtifactCarouselProps) => {
  const freightTimelineControllerContext = useFreightTimelineControllerContext();

  const activeArtifact = selectActiveCarouselFreight(
    props.freight,
    freightTimelineControllerContext?.preferredFilter?.artifactCarousel?.state?.repoURL || ''
  );

  if (typeof activeArtifact === 'string') {
    return 'Invalid Artifact';
  }

  return (
    <div className='flex gap-1 mb-1 items-center justify-between'>
      <Button
        type='text'
        icon={<FontAwesomeIcon icon={faChevronLeft} />}
        onClick={(e) => {
          e.stopPropagation();
          const previousArtifact = selectPreviousArtifact(
            props.freight,
            activeArtifact?.repoURL || ''
          );

          freightTimelineControllerContext?.setPreferredFilter({
            ...freightTimelineControllerContext?.preferredFilter,
            artifactCarousel: {
              ...freightTimelineControllerContext?.preferredFilter?.artifactCarousel,
              state: {
                repoURL: previousArtifact
              }
            }
          });
        }}
      />

      <FreightArtifact artifact={activeArtifact} expand />

      <Button
        type='text'
        icon={<FontAwesomeIcon icon={faChevronRight} />}
        onClick={(e) => {
          e.stopPropagation();
          const nextArtifact = selectNextArtifact(props.freight, activeArtifact?.repoURL || '');

          freightTimelineControllerContext?.setPreferredFilter({
            ...freightTimelineControllerContext?.preferredFilter,
            artifactCarousel: {
              ...freightTimelineControllerContext?.preferredFilter?.artifactCarousel,
              state: {
                repoURL: nextArtifact
              }
            }
          });
        }}
      />
    </div>
  );
};
