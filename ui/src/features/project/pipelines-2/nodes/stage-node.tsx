import { faChevronDown, faFire, faMinus, faTruck } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Dropdown, Flex } from 'antd';
import classNames from 'classnames';
import { formatDistance } from 'date-fns';
import { CSSProperties, ReactNode, useContext, useMemo } from 'react';

import { ColorContext } from '@ui/context/colors';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { StagePhaseIcon } from '@ui/features/common/stage-phase/stage-phase-icon';
import { StagePhase } from '@ui/features/common/stage-phase/utils';
import { ColorMapHex, parseColorAnnotation } from '@ui/features/stage/utils';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import './stage-node.less';
import { useDictionaryContext } from '../context/dictionary-context';
import { useFreightTimelineControllerContext } from '../context/freight-timeline-controller-context';
import { useGraphContext } from '../context/graph-context';
import { stageIndexer } from '../graph/node-indexer';

import style from './node-size-source-of-truth.module.less';
import { StageFreight } from './stage-freight';

export const StageNode = (props: { stage: Stage }) => {
  const dictionaryContext = useDictionaryContext();
  const graphContext = useGraphContext();

  const stageNodeIndex = useMemo(() => stageIndexer.index(props.stage), [props.stage]);

  const headerStyle = useStageHeaderStyle(props.stage);

  const autoPromotionMode =
    dictionaryContext?.stageAutoPromotionMap?.[props.stage?.metadata?.name || ''];

  const stagePhase = getStagePhase(props.stage);
  const stageHealth = getStageHealth(props.stage);

  const controlFlow = isStageControlFlow(props.stage);

  let descriptionItems: ReactNode;

  if (!controlFlow) {
    const lastPromotion = getLastPromotion(props.stage);
    const date = timestampDate(lastPromotion) as Date;

    descriptionItems = (
      <Flex className='text-[10px]' gap={8} wrap vertical>
        <Flex gap={24}>
          <Flex align='center' gap={4}>
            {stagePhase}{' '}
            <StagePhaseIcon
              className={classNames(
                stagePhase !== StagePhase.Promoting && 'text-[8px]',
                stagePhase === StagePhase.Promoting && 'text-[10px] ml-2'
              )}
              noTooltip
              phase={stagePhase}
            />
          </Flex>
          {stageHealth?.status && (
            <Flex gap={4}>
              <Flex align='center' gap={4}>
                {stageHealth?.status}
                <HealthStatusIcon noTooltip className='text-[8px]' health={stageHealth} />
              </Flex>
            </Flex>
          )}
        </Flex>

        {lastPromotion && (
          <Flex gap={4}>
            <span>Last Promotion: </span>
            <span title={date?.toString()}>
              {formatDistance(date, new Date(), { addSuffix: true })}
            </span>
          </Flex>
        )}
      </Flex>
    );
  }

  return (
    <Card
      styles={{
        header: headerStyle,
        body: {
          height: '100%'
        }
      }}
      title={
        <Flex align='center'>
          <span className='text-xs text-wrap'>{props.stage.metadata?.name}</span>
          {autoPromotionMode && (
            <span className='text-[9px] lowercase ml-auto font-normal'>Auto Promotion</span>
          )}
        </Flex>
      }
      className={classNames('stage-node', style['stage-node-size'])}
      size='small'
      variant='borderless'
    >
      {descriptionItems}

      <div className='my-2'>
        <StageFreight stage={props.stage} />
      </div>

      <Dropdown
        trigger={['click']}
        menu={{
          items: [
            {
              label: (
                <span className='text-[10px]'>
                  <FontAwesomeIcon icon={faTruck} className='mr-2' />
                  Promote to Downstream
                </span>
              ),
              key: 'downstream'
            },
            {
              label: (
                <span className='text-[10px] text-orange-600'>
                  <FontAwesomeIcon icon={faFire} className='mr-2' /> Hotfix
                </span>
              ),
              key: 'hotfix'
            }
          ]
        }}
      >
        <Button className='success' size='small'>
          <span>Promote</span>
          <FontAwesomeIcon className='ml-2' icon={faChevronDown} />
        </Button>
      </Dropdown>

      {!graphContext?.stackedNodesParents?.includes(stageNodeIndex) && (
        <Button
          icon={<FontAwesomeIcon icon={faMinus} />}
          size='small'
          className='absolute top-[50%] translate-y-[-50%] text-[10px] z-10'
          onClick={() => graphContext?.onStack(stageNodeIndex)}
        />
      )}
    </Card>
  );
};

const useStageHeaderStyle = (stage: Stage): CSSProperties => {
  const colorContext = useContext(ColorContext);
  if (!useIsColorsUsed()) {
    return {};
  }

  let stageColor =
    parseColorAnnotation(stage) || colorContext.stageColorMap[stage?.metadata?.name || ''];
  let stageFontColor = '';

  if (stageColor && ColorMapHex[stageColor]) {
    stageColor = ColorMapHex[stageColor];
    stageFontColor = 'white';
  }

  return {
    backgroundColor: stageColor || '',
    color: stageFontColor
  };
};

const getStagePhase = (stage: Stage) => stage?.status?.phase as StagePhase;

const isStageControlFlow = (stage: Stage) =>
  (stage?.spec?.promotionTemplate?.spec?.steps?.length || 0) <= 0;

const getStageHealth = (stage: Stage) => stage?.status?.health;

const useIsColorsUsed = () => {
  const freightTimelineControllerContext = useFreightTimelineControllerContext();

  return freightTimelineControllerContext?.preferredFilter?.showColors;
};

const getLastPromotion = (stage: Stage) => stage?.status?.lastPromotion?.finishedAt;
