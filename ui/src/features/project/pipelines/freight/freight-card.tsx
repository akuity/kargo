import { faDocker, faGithub } from '@fortawesome/free-brands-svg-icons';
import {
  faAnchor,
  faCheck,
  faEllipsis,
  faWarehouse,
  IconDefinition
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Divider, Dropdown, Flex, Typography } from 'antd';
import classNames from 'classnames';
import { formatDistance } from 'date-fns';
import { useMemo } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { FreightTimelineControllerContextType } from '@ui/features/project/pipelines/context/freight-timeline-controller-context';
import { ColorMap } from '@ui/features/stage/utils';
import { Freight, Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { FreightArtifact } from './freight-artifact';

type FreightCardProps = {
  freight: Freight;
  viewingFreight?: Freight | null;
  setViewingFreight?(f: Freight | null): void;
  preferredFilter: FreightTimelineControllerContextType['preferredFilter'];
  inUse?: boolean; // is used by stages
  stagesInFreight: Stage[];
  stageColorMap: ColorMap;
  className?: string;
  promote?: boolean;
  soakTime?: string;
  onReviewAndPromote?(): void;
};

export const FreightCard = (props: FreightCardProps) => {
  const navigate = useNavigate();
  const actionContext = useActionContext();

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

  const isViewingFreight =
    props.viewingFreight?.metadata?.name === props.freight?.metadata?.name ||
    actionContext?.action?.freight?.metadata?.name === props.freight?.metadata?.name;

  return (
    <div
      className={classNames(
        'rounded-md text-center flex flex-col cursor-pointer hover:bg-gray-100',
        {
          'bg-gray-50': !isViewingFreight,
          'bg-gray-100': isViewingFreight
        },
        props.className
      )}
      style={{ border: '1px solid rgba(0,0,0,.05)' }}
      onClick={() => {
        navigate(
          generatePath(paths.freight, {
            name: props.freight?.metadata?.namespace,
            freightName: props.freight?.metadata?.name
          })
        );
      }}
    >
      <div
        className={classNames('relative px-2', {
          'pl-4': props.inUse
        })}
      >
        <Flex align='center' justify='space-between' gap={4} className='-mr-2'>
          <Typography.Text className='text-xs text-nowrap' type='secondary'>
            <FontAwesomeIcon className='mr-1' icon={faWarehouse} size='xs' />
            {props.freight?.origin?.name}
          </Typography.Text>
          <Dropdown
            menu={{
              items: [
                {
                  key: 'manually-approve',
                  label: 'Manually Approve',
                  icon: <FontAwesomeIcon icon={faCheck} />,
                  onClick: (e) => {
                    e.domEvent.stopPropagation();
                    actionContext?.actManuallyApprove(props.freight);
                  }
                }
              ]
            }}
          >
            <Button
              icon={<FontAwesomeIcon icon={faEllipsis} />}
              size='small'
              type='text'
              onClick={(e) => e.stopPropagation()}
            />
          </Dropdown>
        </Flex>
        <Divider className='mt-0 mb-1' />

        {props.inUse && !props?.preferredFilter?.showColors && (
          <div className='absolute top-1 bottom-1 left-1 w-1.5 rounded bg-lime-200' />
        )}
        {props?.preferredFilter?.showColors && (
          <div className='flex flex-col gap-0.5 justify-center absolute left-1 top-1 bottom-1'>
            {props.stagesInFreight.map((stage) => (
              <div
                key={stage?.metadata?.uid}
                title={stage?.metadata?.name}
                className='w-1.5 rounded flex-1'
                style={{
                  background: `linear-gradient(60deg,rgb(255 255 255/0%),rgb(200 200 200/30%)), ${props.stageColorMap[stage?.metadata?.name || '']}`
                }}
              />
            ))}
          </div>
        )}

        {props?.preferredFilter?.showAlias && (
          <div className='text-[10px] text-nowrap mb-2'>{freightAlias}</div>
        )}

        <div className='flex flex-col gap-1 justify-center items-center'>
          {props.freight?.commits
            ?.slice(0, 2)
            .map((commit) => <FreightArtifact key={commit?.repoURL} artifact={commit} />)}

          {props.freight?.charts
            ?.slice(0, 2)
            .map((chart) => <FreightArtifact key={chart?.repoURL} artifact={chart} />)}

          {props.freight?.images
            ?.slice(0, 2)
            .map((image) => <FreightArtifact key={image?.repoURL} artifact={image} />)}

          {noOfGitCommits + noOfHelmReleases + noOfContainerImages > 6 && (
            <Typography.Text type='secondary' className='text-[10px]'>
              +
              {noOfGitCommits +
                noOfHelmReleases +
                noOfContainerImages -
                (props.freight?.charts?.slice(0, 2)?.length +
                  props.freight?.charts?.slice(0, 2)?.length +
                  props.freight?.images?.slice(0, 2)?.length)}{' '}
              more
            </Typography.Text>
          )}
        </div>

        <div className='flex flex-col mx-auto w-full gap-0.5 items-center justify-center text-nowrap my-1'>
          {(noOfGitCommits ? 1 : 0) + (noOfHelmReleases ? 1 : 0) + (noOfContainerImages ? 1 : 0) >
            2 && (
            <>
              <FreightCard.ArtifactCount icon={faGithub} count={noOfGitCommits} />

              <FreightCard.ArtifactCount icon={faAnchor} count={noOfHelmReleases} />

              <FreightCard.ArtifactCount icon={faDocker} count={noOfContainerImages} />
            </>
          )}
          <Typography.Text
            className='text-xs text-nowrap'
            type='secondary'
            title={creation.abs?.toString()}
          >
            {creation.relative}
          </Typography.Text>
        </div>
      </div>

      {props.promote && (
        <div className='px-1 pb-1'>
          <Button
            className='w-full'
            type='primary'
            size='small'
            onClick={(e) => {
              e.stopPropagation();
              props.onReviewAndPromote?.();
            }}
          >
            Select
          </Button>
        </div>
      )}
    </div>
  );
};

FreightCard.ArtifactCount = (props: { icon: IconDefinition; count: number }) =>
  props.count > 0 && (
    <div className='text-[10px]'>
      {props.count}x <FontAwesomeIcon icon={props.icon} />
    </div>
  );
