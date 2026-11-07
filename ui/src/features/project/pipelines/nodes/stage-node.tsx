import { useDroppable } from '@dnd-kit/core';
import {
  faEllipsisVertical,
  faMinus,
  faShareNodes,
  faTruckArrowRight
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Dropdown, Flex, message } from 'antd';
import { ItemType } from 'antd/es/menu/interface';
import classNames from 'classnames';
import { formatDistance } from 'date-fns';
import { CSSProperties, useMemo } from 'react';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { useArgoCDMenuItems } from '@ui/features/project/pipelines/nodes/use-argocd-links';
import { useApproveFreight } from '@ui/gen/api/v2/core/core';
import { Stage } from '@ui/gen/api/v2/models';

import { useDictionaryContext } from '../context/dictionary-context';
import { useGraphContext } from '../context/graph-context';
import { stageIndexer } from '../graph/node-indexer';
import { AutoPromotionStatusIcon } from '../promotion/auto-promotion-status-icon';
import { DropOverlay } from '../promotion/drag-and-drop/drop-overlay';
import { isPromotionWindowClosed } from '../promotion/promotion-window';

import { AnalysisRunLogsLink } from './analysis-run-logs-link';
import style from './node-size-source-of-truth.module.less';
import { useGetPromotionDropdownItems } from './promotion/use-get-promotion-dropdown-items';
import { StageFreight } from './stage-freight';
import {
  getCurrentPromotionRef,
  getLastPromotionDate,
  getLastPromotionRef,
  isStageControlFlow,
  useHideStageIfInPromotionMode,
  useStageAccentColor
} from './stage-meta-utils';
import { StageNodeHealth } from './stage-node-health';
import { StageNodePhase } from './stage-node-phase';
import { StageNodeWaiting } from './stage-node-waiting';
import { useStageWaiting } from './use-stage-waiting';

import './stage-node.less';

export const StageNode = (props: { stage: Stage }) => {
  const stageName = props.stage?.metadata?.name || '';
  const projectName = props.stage?.metadata?.namespace || '';

  const dictionaryContext = useDictionaryContext();
  const graphContext = useGraphContext();
  const actionContext = useActionContext();

  const stageNodeIndex = useMemo(() => stageIndexer.index(props.stage), [props.stage]);

  const accentColor = useStageAccentColor(props.stage);

  const autoPromotionMode = dictionaryContext?.stageAutoPromotionMap?.[stageName];

  const controlFlow = isStageControlFlow(props.stage);

  // A Stage with a color of its own states it twice: outright in the thicker
  // top border, and as a wash behind the header. That border takes 3px more
  // than a plain one, which the body's top padding gives back so the content
  // sits in the same place either way -- and which the header is told about,
  // so its wash reaches the border instead of stopping short of it.
  const accentBorder = Boolean(accentColor) && !controlFlow;
  const bodyPaddingTop = accentBorder ? 8 : 11;

  const hideStage = useHideStageIfInPromotionMode(props.stage);

  const totalSubscribersToThisStage = dictionaryContext?.subscribersByStage?.[stageName]?.size || 0;

  const manualApproveActionMutation = useApproveFreight({
    mutation: {
      onSuccess: (_, vars) => {
        message.success(
          `Freight ${actionContext?.action?.freight?.alias} has been manually approved for stage ${vars.params.stage}`
        );

        actionContext?.cancel();
      }
    }
  });

  const { dropdownItems, resumeAutoPromotionDrawer } = useGetPromotionDropdownItems(props.stage);

  const argoCDMenuItems = useArgoCDMenuItems(props.stage);

  // More than one way to promote folds into a submenu, so the menu reads as
  // one line per kind of action. A Stage with a single way keeps it in place.
  const promotionItems: ItemType[] =
    dropdownItems.length > 1
      ? [
          {
            key: 'promote-menu',
            icon: (
              <span className='stage-node__menu-icon'>
                <FontAwesomeIcon icon={faTruckArrowRight} className='text-[11px]' />
              </span>
            ),
            label: 'Promote',
            children: dropdownItems
          }
        ]
      : dropdownItems;

  // One menu for everything the node can do: the Stage's Argo CD Applications
  // first, the promotion actions after a divider.
  const menuItems: ItemType[] = [
    ...(argoCDMenuItems.length
      ? [...argoCDMenuItems, { type: 'divider' as const, key: 'argocd-divider' }]
      : []),
    ...promotionItems
  ];

  const currentPromotion = getCurrentPromotionRef(props.stage);

  // What, if anything, has stopped this Stage's promotion and is waiting on a
  // person to deal with it.
  const waiting = useStageWaiting(props.stage);

  const isPromoting = Boolean(currentPromotion);

  const lastPromotion = getLastPromotionDate(props.stage);
  const lastPromotionPath = getLastPromotionRef(props.stage)?.path;

  // The footer is the date of the last promotion and nothing else now that
  // the waiting tags have moved up into the status band, so a Stage that has
  // never been promoted to has nothing to put there. Rendering the row anyway
  // left its top margin hanging under the card as a stray gap.
  const showFooter = Boolean(lastPromotion && lastPromotionPath);

  const stagePath = generatePath(paths.stage, { name: projectName, stageName });

  const { isOver, setNodeRef } = useDroppable({
    id: props.stage.metadata?.name || 'stage-node',
    data: {
      requestedFreightNames: props.stage.spec?.requestedFreight?.map((f) => f.origin?.name) || [],
      promotionWindowClosed: isPromotionWindowClosed(props.stage)
    }
  });

  const isSearchMatch = Boolean(graphContext?.matchedStageIndices?.has(stageNodeIndex));

  return (
    <div
      ref={setNodeRef}
      style={{
        transform: isOver ? 'scale(0.98)' : undefined,
        transition: 'transform 0.1s ease, opacity 0.2s ease'
      }}
    >
      <Card
        // AntD's own card styles match these selectors, so radius and padding
        // are set here rather than lost to a specificity tie in the
        // stylesheet.
        //
        // The Stage's color is the card's own top border rather than a strip
        // laid inside it. A border follows the card's corners exactly, so
        // there is nothing to align and nothing to clip.
        style={
          {
            borderRadius: 10,
            borderTop: accentBorder ? `4px solid ${accentColor}` : undefined,
            // The color the header washes itself with. Left unset -- a control
            // flow Stage, or any Stage at all with colors turned off -- the
            // wash's color-mix has nothing to resolve and the header stays
            // flat.
            '--stage-accent': accentBorder ? accentColor : undefined,
            '--stage-head-pull': `${bodyPaddingTop}px`
          } as CSSProperties
        }
        styles={{
          body: {
            padding: `${bodyPaddingTop}px 10px 10px`,
            height: '100%',
            position: 'relative'
          }
        }}
        className={classNames('stage-node', style['stage-node-size'], {
          'opacity-40': hideStage,
          'stage-node--control-flow': controlFlow,
          // no color to band the header with, so it is ruled off instead
          'stage-node--uncolored': !accentBorder,
          'stage-node--promoting': isPromoting,
          'stage-node--search-match': isSearchMatch
        })}
        size='small'
      >
        <DropOverlay isOver={isOver} stage={props.stage} />

        <Flex align='center' gap={8} className='stage-node__header min-w-0'>
          {/* everything up to the action buttons opens the Stage */}
          <Link to={stagePath} className='stage-node__title'>
            {controlFlow ? (
              <span className='stage-node__dot stage-node__dot--muted' />
            ) : (
              accentColor && (
                <span className='stage-node__dot' style={{ backgroundColor: accentColor }} />
              )
            )}

            <div className='min-w-0 flex-1'>
              <Flex align='center' className='min-w-0'>
                <AutoPromotionStatusIcon
                  stage={props.stage}
                  autoPromotionEnabled={Boolean(autoPromotionMode)}
                />
                <span className='stage-node__name' title={stageName}>
                  {stageName}
                </span>
              </Flex>
            </div>
          </Link>

          <Dropdown trigger={['click']} classNames={{ root: 'w-fit' }} menu={{ items: menuItems }}>
            <Button
              size='small'
              type='text'
              className='stage-node__action flex-none'
              title='Actions'
              icon={<FontAwesomeIcon icon={faEllipsisVertical} />}
            />
          </Dropdown>
        </Flex>

        {/* a control flow Stage has no steps of its own, so the label is all
            the card has to say below its header */}
        {controlFlow && (
          <div className='stage-node__control-flow-label'>
            <FontAwesomeIcon icon={faShareNodes} className='stage-node__control-flow-icon' />
            Control flow
          </div>
        )}

        {!controlFlow && (
          <div className='stage-node__status-grid'>
            <div className='stage-node__status-cell'>
              <div className='stage-node__status-label'>Status</div>

              {/* A promotion stopped on a step says so here rather than in a
                  tag below: "Promoting" is true but useless once something is
                  waiting on a person, and what they need is which thing, with
                  the way to go and deal with it.

                  The analysis run is how a Stage verifies and the phase is
                  what reports the verifying, so its logs belong here too,
                  rather than off with the other links. */}
              {/* the gaps are the stylesheet's: a numeric `gap` here would be
                  written inline and take the rows with it */}
              <Flex
                align='center'
                wrap
                className='stage-node__status stage-node__status-line min-w-0'
              >
                {waiting ? (
                  <StageNodeWaiting waiting={waiting} />
                ) : (
                  <>
                    <StageNodePhase stage={props.stage} />

                    {dictionaryContext?.hasAnalysisRunLogsUrlTemplate && (
                      <AnalysisRunLogsLink stage={props.stage} />
                    )}
                  </>
                )}
              </Flex>
            </div>

            <div className='stage-node__status-cell'>
              <div className='stage-node__status-label'>Health</div>
              <StageNodeHealth stage={props.stage} />
            </div>
          </div>
        )}

        {actionContext?.action?.type === IAction.MANUALLY_APPROVE ? (
          <Button
            className='success'
            size='small'
            loading={manualApproveActionMutation.isPending}
            onClick={() => {
              manualApproveActionMutation.mutate({
                params: {
                  stage: stageName
                },
                project: projectName,
                freightNameOrAlias: actionContext?.action?.freight?.metadata?.name || ''
              });
            }}
          >
            Approve
          </Button>
        ) : (
          !controlFlow && <StageFreight stage={props.stage} />
        )}

        {!controlFlow && showFooter && (
          <div className='stage-node__footer mt-2 min-w-0'>
            {/* the last thing the Stage did. What it is doing now -- and what
                it is waiting on -- is stated once, by the Status above. */}
            <Link
              to={lastPromotionPath!}
              className='stage-node__footer-link'
              title={lastPromotion!.toString()}
            >
              Promoted{' '}
              <span className='stage-node__footer-time'>
                {formatDistance(lastPromotion!, new Date(), { addSuffix: true })}
              </span>
            </Link>
          </div>
        )}
      </Card>

      {resumeAutoPromotionDrawer}

      {!graphContext?.stackedNodesParents?.includes(stageNodeIndex) &&
        totalSubscribersToThisStage > 0 && (
          <Button
            style={{ width: 16, height: 16 }}
            icon={<FontAwesomeIcon icon={faMinus} />}
            size='small'
            className='absolute top-[50%] right-0 translate-x-[50%] translate-y-[-50%] text-[8px] z-10'
            onClick={() => graphContext?.onStack(stageNodeIndex)}
          />
        )}
    </div>
  );
};
