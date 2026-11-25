import { useDraggable } from '@dnd-kit/core';
import {
  faCheck,
  faEllipsisVertical,
  faGrip,
  faHourglass,
  faPlus,
  faTrash,
  faTriangleExclamation
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Dropdown, Flex, Tooltip } from 'antd';
import classNames from 'classnames';
import { Duration, formatDistance, formatDuration } from 'date-fns';
import { CSSProperties, useEffect, useMemo, useRef } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { RepoSubscription } from '@ui/extend/types';
import { useModal } from '@ui/features/common/modal/use-modal';
import { getShortFreightLabel } from '@ui/features/common/utils';
import { useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { FreightTimelineControllerContextType } from '@ui/features/project/pipelines/context/freight-timeline-controller-context';
import { ColorMap } from '@ui/features/stage/utils';
import { Freight, Stage } from '@ui/gen/api/v2/models';

import { useManualApprovalModal } from '../promotion/use-manual-approval-modal';

import { DeleteFreightModal } from './delete-freight-modal';
import { FreightArtifactList } from './freight-artifact-list';
import { FreightStageBar } from './freight-stage-bar';
import { getFreightDiscoveredAt } from './get-freight-discovered-at';
import { OverflowText } from './overflow-text';
import { useSoakTimeCounter } from './use-soak-time-counter';

// Ring and border color for the selected card, from the freightline design.
const HIGHLIGHT_COLOR = '#4f8cff';

// Secondary text (age, grip), from the freightline design.
const MUTED_TEXT_CLASS = 'text-[#8c8c8c]';

// Footer controls are squeezed into the card's 18px footer row.
const footerButtonStyle: CSSProperties = {
  height: 18,
  fontSize: 11,
  paddingInline: 4,
  borderRadius: 4
};

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
  // the piece of freight its Warehouse discovered before this one, which the
  // artifacts are compared with to pick out what differs
  previousFreight?: Freight;
  // whether the artifacts name the Warehouse subscription they came from
  showSubscriptionNames?: boolean;
  // the subscriptions of the freight's Warehouse, to name the ones its
  // artifacts do not record
  subscriptions?: RepoSubscription[];
};

export const FreightCard = (props: FreightCardProps) => {
  const navigate = useNavigate();
  const actionContext = useActionContext();

  const freightName = props.freight?.metadata?.name || '';
  const freightAlias = props.freight?.alias;

  const deleteFreightModal = useModal();

  const creation = useMemo(() => {
    const creationDate = getFreightDiscoveredAt(props.freight);

    if (!creationDate) {
      return { relative: '', abs: null };
    }

    return {
      relative: formatDistance(creationDate, new Date(), { addSuffix: false })
        ?.replace('about', '')
        .trim(),
      abs: creationDate
    };
  }, [props.freight]);

  const isViewingFreight =
    props.viewingFreight?.metadata?.name === freightName ||
    actionContext?.action?.freight?.metadata?.name === freightName;

  const soakTime = useSoakTimeCounter(props.soakTime, props.freight?.metadata?.namespace);

  const frozenInitialSoakTime = useRef(props.soakTime);

  useEffect(() => {
    if (!frozenInitialSoakTime.current) {
      frozenInitialSoakTime.current = props.soakTime;
    }
  }, [props.soakTime]);

  const soakTimeFormatted = useMemo(() => (soakTime ? formatDuration(soakTime) : ''), [soakTime]);

  const { attributes, listeners, setNodeRef, transform } = useDraggable({
    id: freightName || 'name',
    data: { originName: props.freight.origin?.name }
  });

  const showManualApproveModal = useManualApprovalModal();

  const dragStyle: CSSProperties | undefined = transform
    ? {
        transform: `translate3d(${transform.x}px, ${transform.y}px, 0) scale(1.05)`,
        zIndex: 9999,
        boxShadow: '0 4px 8px rgba(0,0,0,.1)'
      }
    : undefined;

  const highlightStyle: CSSProperties | undefined = isViewingFreight
    ? { borderColor: HIGHLIGHT_COLOR, boxShadow: `0 0 0 1px ${HIGHLIGHT_COLOR}` }
    : undefined;

  return (
    <div
      ref={setNodeRef}
      style={{ width: 'var(--freight-card-width, 150px)', ...dragStyle }}
      className='relative shrink-0 flex'
    >
      <div
        className={classNames(
          'relative flex-1 min-w-0 flex flex-col overflow-hidden rounded-md border border-solid cursor-pointer py-1.5 pr-2 transition-colors',
          // the stage bar takes the left edge of a held card; without it, the
          // card is padded evenly
          props.inUse ? 'pl-3' : 'pl-2',
          // Card colors from the freightline design; they sit on the page
          // background so cards read as raised from the strip. Every card
          // looks the same: the stage bar alone marks freight a Stage holds.
          'bg-white border-gray-300 shadow-sm dark:shadow-none dark:bg-[#262626] dark:border-[#3d3d3d]',
          // one step past the card's border, so hovering shows without
          // outshining the selected highlight
          !highlightStyle && 'hover:border-[#c4c9d0] dark:hover:border-[#4d4d4d]',
          props.className
        )}
        style={highlightStyle}
        onClick={(e) => {
          // React passes up clicks from the card's tooltips and menus, which
          // are drawn elsewhere in the page; only a click on the card itself
          // opens the freight.
          if (!e.currentTarget.contains(e.target as Node)) {
            return;
          }

          navigate(
            generatePath(paths.freight, {
              name: props.freight?.metadata?.namespace,
              freightName
            })
          );
        }}
      >
        <FreightStageBar
          stages={props.stagesInFreight}
          stageColorMap={props.stageColorMap}
          showColors={props.preferredFilter?.showColors}
        />

        {/* The alias is what a user recognizes a piece of freight by; the
            head of the generated name takes its place when aliases are turned
            off. The Warehouse is named once above its run of freight, not on
            every card. */}
        <Flex align='center' gap={4} className='h-5 min-w-0'>
          <OverflowText
            text={
              (props.preferredFilter?.showAlias && freightAlias) ||
              getShortFreightLabel(freightName)
            }
            title={freightName}
            className='flex-1 min-w-0 text-[12px] leading-5'
          />
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
            {/* A plain button rather than AntD's, so it has no hover fill. It
                reaches past the header row and into the card's right padding
                to make a larger target, and is pulled left over the header gap
                so the dots sit close to the alias. */}
            <button
              type='button'
              aria-label='Freight actions'
              className='shrink-0 flex items-center justify-center w-6 h-7 -my-1 -ml-2 -mr-2 p-0 border-0 bg-transparent cursor-pointer text-[color:var(--kargo-color-text)]'
              onClick={(e) => e.stopPropagation()}
            >
              <FontAwesomeIcon icon={faEllipsisVertical} className='text-[12px]' />
            </button>
          </Dropdown>
        </Flex>

        {/* at least two artifact rows tall (2 x 20px + 2px gap), so a card with
            one artifact matches the height of one with two */}
        <div className='flex flex-col gap-[2px] mt-1 min-w-0 min-h-[42px]'>
          <FreightArtifactList
            freight={props.freight}
            previousFreight={props.previousFreight}
            showSubscriptionNames={props.showSubscriptionNames}
            subscriptions={props.subscriptions}
          />
        </div>

        <div
          className='mt-auto pt-1 min-w-0'
          // While promoting, the footer is the Select button's, and a click
          // there never opens the freight -- not even while the button is
          // loading, when AntD drops the click before its handler sees it.
          onClick={props.promote ? (e) => e.stopPropagation() : undefined}
        >
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
                style={footerButtonStyle}
                type='primary'
                size='small'
                onClick={() => {
                  const freight = freightName;
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
                    <FontAwesomeIcon
                      icon={soakTimeFormatted ? faHourglass : faTriangleExclamation}
                    />
                  ) : undefined
                }
              >
                Select
              </Button>
            </Tooltip>
          ) : (
            <Flex align='center' justify='space-between' gap={4} className='h-[18px] min-w-0'>
              <span
                className={classNames('truncate text-[11px] leading-[18px]', MUTED_TEXT_CLASS)}
                title={creation.abs?.toString()}
              >
                {creation.relative}
              </span>
              <div
                {...listeners}
                {...attributes}
                className={classNames(
                  'shrink-0 flex items-center justify-center w-7 h-6 -my-[3px] -mr-1.5 cursor-grab active:cursor-grabbing',
                  MUTED_TEXT_CLASS
                )}
                onClick={(e) => e.stopPropagation()}
                onMouseEnter={(e) => e.stopPropagation()}
              >
                <FontAwesomeIcon icon={faGrip} className='text-[11px]' />
              </div>
            </Flex>
          )}
        </div>
      </div>
    </div>
  );
};
