import { useDraggable } from '@dnd-kit/core';
import { faDocker, faGithub } from '@fortawesome/free-brands-svg-icons';
import {
  faAnchor,
  faCheck,
  faEllipsis,
  faPlus,
  faGripVertical,
  faHourglass,
  faTrash,
  faTriangleExclamation,
  faWarehouse,
  IconDefinition
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Divider, Dropdown, Flex, Tooltip, Typography } from 'antd';
import classNames from 'classnames';
import { Duration, formatDistance, formatDuration } from 'date-fns';
import { useEffect, useMemo, useRef } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useModal } from '@ui/features/common/modal/use-modal';
import { useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { FreightTimelineControllerContextType } from '@ui/features/project/pipelines/context/freight-timeline-controller-context';
import { ColorMap } from '@ui/features/stage/utils';
import { Freight, Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { useManualApprovalModal } from '../promotion/use-manual-approval-modal';

import { DeleteFreightModal } from './delete-freight-modal';
import { FreightArtifact } from './freight-artifact';
import { useSoakTimeCounter } from './use-soak-time-counter';

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
  onReviewAndPromote?(): void;
  soakTime?: Duration;
  promotionEligible?: boolean;
  isPromotionEligibleLoading?: boolean;
};

export const FreightCard = (props: FreightCardProps) => {
  const navigate = useNavigate();
  const actionContext = useActionContext();

  const freightAlias = props.freight?.alias;

  const deleteFreightModal = useModal();

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

  const soakTime = useSoakTimeCounter(props.soakTime);

  const frozenInitialSoakTime = useRef(props.soakTime);

  useEffect(() => {
    if (!frozenInitialSoakTime.current) {
      frozenInitialSoakTime.current = props.soakTime;
    }
  }, [props.soakTime]);

  const soakTimeFormatted = useMemo(() => (soakTime ? formatDuration(soakTime) : ''), [soakTime]);

  const { attributes, listeners, setNodeRef, transform } = useDraggable({
    id: props.freight.metadata?.name || 'name',
    data: { originName: props.freight.origin?.name }
  });

  const showManualApproveModal = useManualApprovalModal();

  const style = transform
    ? {
        transform: `translate3d(${transform.x}px, ${transform.y}px, 0) scale(1.05)`,
        zIndex: 9999,
        boxShadow: '0 4px 8px rgba(0,0,0,.1)'
      }
    : undefined;

  return (
    <div ref={setNodeRef} style={style} className='relative'>
      <div className='absolute bottom-0 left-0 right-0 p-1'>
        {props.promote ? (
          <Tooltip
            title={
              !props.promotionEligible && !props.isPromotionEligibleLoading
                ? soakTimeFormatted
                  ? `Soak: ${soakTimeFormatted}`
                  : 'Non-approved freight'
                : ''
            }
          >
            <Button
              className={classNames('w-full', { 'opacity-70': !props.promotionEligible })}
              type='primary'
              size='small'
              onClick={() => {
                const freight = props.freight?.metadata?.name || '';
                const stage = actionContext?.action?.stage?.metadata?.name || '';
                const projectName = props.freight?.metadata?.namespace || '';

                if (props.promotionEligible) {
                  props.onReviewAndPromote?.();
                  return;
                }

                showManualApproveModal({
                  freight,
                  stage,
                  projectName,
                  onApprove: () => {
                    navigate(
                      generatePath(paths.promote, {
                        name: projectName,
                        freight: freight,
                        stage
                      })
                    );
                  }
                });
              }}
              loading={props.isPromotionEligibleLoading}
              icon={
                !props.promotionEligible && !props.isPromotionEligibleLoading ? (
                  <FontAwesomeIcon icon={soakTimeFormatted ? faHourglass : faTriangleExclamation} />
                ) : undefined
              }
            >
              Select
            </Button>
          </Tooltip>
        ) : (
          <div
            {...listeners}
            {...attributes}
            className='bg-gray-100 rounded text-center cursor-pointer hover:bg-gray-200 active:bg-gray-200'
            style={{ padding: '3px 0 1px' }}
            onMouseEnter={(e) => e.stopPropagation()}
          >
            <FontAwesomeIcon icon={faGripVertical} className='text-gray-500' size='sm' />
          </div>
        )}
      </div>
      <div
        className={classNames(
          'rounded-md text-center flex flex-col cursor-pointer pb-7 border border-solid border-gray-100 hover:border-gray-300',
          {
            'bg-gray-50': !isViewingFreight,
            'bg-gray-100': isViewingFreight
          },
          props.className
        )}
        onClick={() =>
          navigate(
            generatePath(paths.freight, {
              name: props.freight?.metadata?.namespace,
              freightName: props.freight?.metadata?.name
            })
          )
        }
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
                    key: 'similar-freight',
                    label: 'Clone freight',
                    icon: <FontAwesomeIcon icon={faPlus} />,
                    onClick: (e) => {
                      e.domEvent.stopPropagation();
                      navigate(
                        `${generatePath(paths.warehouse, {
                          name: props.freight?.metadata?.namespace,
                          warehouseName: props.freight?.origin?.name,
                          tab: 'create-freight'
                        })}?clone-freight=${props.freight?.alias}`
                      );
                    }
                  },
                  {
                    key: 'manually-approve',
                    label: 'Manually Approve',
                    icon: <FontAwesomeIcon icon={faCheck} />,
                    onClick: (e) => {
                      e.domEvent.stopPropagation();
                      actionContext?.actManuallyApprove(props.freight);
                    }
                  },
                  {
                    key: 'delete-freight',
                    label: 'Delete Freight',
                    icon: <FontAwesomeIcon icon={faTrash} />,
                    onClick: (e) => {
                      e.domEvent.stopPropagation();
                      deleteFreightModal.show((modalProps) => (
                        <DeleteFreightModal
                          freight={props.freight}
                          onDelete={modalProps.hide}
                          {...modalProps}
                        />
                      ));
                    },
                    disabled: props.inUse
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
            {props.freight?.commits?.slice(0, 2).map((commit) => (
              <FreightArtifact key={commit?.repoURL} artifact={commit} />
            ))}

            {props.freight?.charts?.slice(0, 2).map((chart) => (
              <FreightArtifact key={chart?.repoURL} artifact={chart} />
            ))}

            {props.freight?.images?.slice(0, 2).map((image) => (
              <FreightArtifact key={image?.repoURL} artifact={image} />
            ))}

            {props.freight?.artifacts?.slice(0, 2).map((other) => (
              <FreightArtifact key={other?.version} artifact={other} />
            ))}

            {noOfGitCommits + noOfHelmReleases + noOfContainerImages > 6 && (
              <Typography.Text type='secondary' className='text-[10px]'>
                +
                {noOfGitCommits +
                  noOfHelmReleases +
                  noOfContainerImages -
                  (props.freight?.charts?.slice(0, 2)?.length +
                    props.freight?.commits?.slice(0, 2)?.length +
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
