import { faDocker, faGithub } from '@fortawesome/free-brands-svg-icons';
import { faAnchor, faWarehouse, IconDefinition } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tag, Typography } from 'antd';
import classNames from 'classnames';
import { formatDistance } from 'date-fns';
import { useMemo } from 'react';

import { FreightTimelineControllerContextType } from '@ui/features/project/pipelines-2/context/freight-timeline-controller-context';
import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { FreightArtifact } from './freight-artifact';
import { FreightArtifactCarousel } from './freight-artifacts-carousel';

type FreightCardProps = {
  freight: Freight;
  viewingFreight?: Freight | null;
  setViewingFreight?(f: Freight | null): void;
  preferredFilter: FreightTimelineControllerContextType['preferredFilter'];
  inUse?: boolean; // is used by stages
};

export const FreightCard = (props: FreightCardProps) => {
  const freightAlias = props.freight?.alias;

  const creation = useMemo(() => {
    const creationDate = timestampDate(props.freight?.metadata?.creationTimestamp);

    if (!creationDate) {
      return { relative: '', abs: creationDate };
    }

    return {
      relative: formatDistance(creationDate, new Date(), { addSuffix: false })?.replace(
        'about',
        ''
      ),
      abs: creationDate
    };
  }, [props.freight]);

  const noOfGitCommits = props.freight?.commits?.length || 0;
  const noOfHelmReleases = props.freight?.charts?.length || 0;
  const noOfContainerImages = props.freight?.images?.length || 0;

  const isViewingFreight = props.viewingFreight?.metadata?.name === props.freight?.metadata?.name;

  return (
    <div
      className={classNames(
        'pt-2 px-2 rounded-md text-center flex flex-col cursor-pointer hover:bg-gray-100 relative justify-center',
        {
          'bg-gray-50': !isViewingFreight,
          'bg-gray-100': isViewingFreight
        }
      )}
      style={{ border: '1px solid rgba(0,0,0,.05)' }}
      onClick={() => props.setViewingFreight?.(isViewingFreight ? null : props.freight)}
    >
      {props.inUse && (
        <Tag
          className='w-fit text-[8px] absolute right-[-20px] top-0 leading-none'
          bordered={false}
          color='green'
        >
          in use
        </Tag>
      )}
      {props?.preferredFilter?.showColors && (
        <div className='flex gap-1 mb-1 justify-center'>
          <div
            title='dev'
            className='mx-1 h-3 w-3 rounded'
            style={{
              background: '#45B084'
            }}
          />
        </div>
      )}

      {props?.preferredFilter?.showAlias && (
        <div className='text-[10px] text-nowrap mb-2'>{freightAlias}</div>
      )}

      {!props?.preferredFilter?.artifactCarousel?.enabled && (
        <div className='flex gap-1 justify-center'>
          {props.freight?.commits?.map((commit) => (
            <FreightArtifact key={commit?.repoURL} artifact={commit} />
          ))}

          {props.freight?.charts?.map((chart) => (
            <FreightArtifact key={chart?.repoURL} artifact={chart} />
          ))}

          {props.freight?.images?.map((image) => (
            <FreightArtifact key={image?.repoURL} artifact={image} />
          ))}
        </div>
      )}

      {props?.preferredFilter?.artifactCarousel?.enabled && (
        <FreightArtifactCarousel freight={props.freight} />
      )}

      <div className='flex mx-auto w-full gap-2 items-center justify-center text-nowrap mt-1'>
        {noOfGitCommits + noOfHelmReleases + noOfContainerImages > 0 && (
          <>
            <FreightCard.ArtifactCount icon={faGithub} count={noOfGitCommits} />

            <FreightCard.ArtifactCount icon={faAnchor} count={noOfHelmReleases} />

            <FreightCard.ArtifactCount icon={faDocker} count={noOfContainerImages} />
          </>
        )}
        <Typography.Text
          className='text-[10px] text-nowrap'
          type='secondary'
          title={creation.abs?.toString()}
        >
          {creation.relative}
        </Typography.Text>
        <Typography.Text className='text-[10px] text-nowrap' type='secondary'>
          <FontAwesomeIcon className='mr-1' icon={faWarehouse} />
          {props.freight?.origin?.name}
        </Typography.Text>
      </div>
    </div>
  );
};

FreightCard.ArtifactCount = (props: { icon: IconDefinition; count: number }) =>
  props.count > 0 && (
    <div className='text-[10px]'>
      {props.count}x <FontAwesomeIcon icon={props.icon} />
    </div>
  );
