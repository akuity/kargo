import {
  faChevronDown,
  faChevronLeft,
  faChevronRight,
  faCodeCommit,
  faExternalLink,
  faFire,
  faTruck
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Dropdown, Flex, Tag, Typography } from 'antd';
import Link from 'antd/es/typography/Link';
import classNames from 'classnames';
import { formatDistance } from 'date-fns';
import { CSSProperties, ReactNode, useEffect, useMemo, useState } from 'react';

import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { StagePhaseIcon } from '@ui/features/common/stage-phase/stage-phase-icon';
import { StagePhase } from '@ui/features/common/stage-phase/utils';
import { getCurrentFreight } from '@ui/features/common/utils';
import {
  getGitCommitURL,
  getImageBuiltDate,
  getImageSource
} from '@ui/features/freight-timeline/open-container-initiative-utils';
import { ColorMapHex, parseColorAnnotation } from '@ui/features/stage/utils';
import {
  Chart,
  FreightReference,
  GitCommit,
  Image,
  Stage
} from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import './stage-node.less';
import { humanComprehendableArtifact } from '../freight/artifact-parts-utils';
import { shortVersion } from '../freight/short-version-utils';

import {
  ArtifactTypes,
  selectFirstArtifact,
  selectNextArtifact,
  selectPreviousArtifact
} from './artifact-selector-utils';
import style from './node-size-source-of-truth.module.less';

export const StageNode = (props: { stage: Stage }) => {
  const headerStyle = useStageHeaderStyle(props.stage);

  const autoPromotionMode = useIsStageAutoPromotionMode();

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
    </Card>
  );
};

const StageFreight = (props: { stage: Stage }) => {
  const currentFreight = useMemo(() => getCurrentFreight(props.stage), [props.stage]);

  const warehouses = currentFreight?.map((f) => f.origin?.name);

  const [selectedWarehouse, setSelectedWarehouse] = useState(warehouses?.[0]);

  const defaultToFirstFreight = () =>
    currentFreight?.find((f) => f?.origin?.name === selectedWarehouse) as FreightReference;

  const [selectedFreight, setSelectedFreight] = useState(defaultToFirstFreight);

  useEffect(() => setSelectedFreight(defaultToFirstFreight()), [selectedWarehouse]);

  const defaultToFirstArtifact = () =>
    // @ts-expect-error FreightReference and Freight are same, at least in this case
    selectFirstArtifact([selectedFreight]) as ArtifactTypes;

  const [selectedArtifact, setSelectedArtifact] = useState(defaultToFirstArtifact());

  useEffect(() => setSelectedArtifact(defaultToFirstArtifact()), [selectedFreight]);

  const onNextArtifact = () => {
    setSelectedArtifact(selectNextArtifact(selectedFreight, selectedArtifact));
  };

  const onPreviousArtifact = () => {
    setSelectedArtifact(selectPreviousArtifact(selectedFreight, selectedArtifact));
  };

  const onNextWarehouse = () => {
    const currentWarehouseIndex = warehouses.findIndex((w) => w === selectedWarehouse);

    const nextWarehouseIndex = (currentWarehouseIndex + 1) % warehouses?.length;

    setSelectedWarehouse(warehouses[nextWarehouseIndex]);
  };

  const onPreviousWarehouse = () => {
    const currentWarehouseIndex = warehouses.findIndex((w) => w === selectedWarehouse);

    let nextWarehouseIndex = currentWarehouseIndex - 1;

    if (nextWarehouseIndex < 0) {
      nextWarehouseIndex = warehouses.length - 1;
    }

    setSelectedWarehouse(warehouses[nextWarehouseIndex]);
  };

  if (!currentFreight?.length) {
    return null;
  }

  const noOfGitCommits = selectedFreight?.commits?.length || 0;
  const noOfHelmReleases = selectedFreight?.charts?.length || 0;
  const noOfContainerImages = selectedFreight?.images?.length || 0;

  const totalArtifacts = noOfContainerImages + noOfGitCommits + noOfHelmReleases;

  return (
    <>
      {warehouses?.length > 1 && (
        <Flex align='center' justify='center' className='text-[10px] font-light'>
          <Button
            icon={<FontAwesomeIcon icon={faChevronLeft} />}
            size='small'
            type='text'
            className='mr-auto text-[10px]'
            onClick={onPreviousWarehouse}
          />
          {selectedWarehouse}
          <Button
            icon={<FontAwesomeIcon icon={faChevronRight} />}
            size='small'
            type='text'
            className='ml-auto text-[10px]'
            onClick={onNextWarehouse}
          />
        </Flex>
      )}

      <Flex align='center' justify='center' className='my-2'>
        {totalArtifacts > 1 && (
          <Button
            icon={<FontAwesomeIcon icon={faChevronLeft} />}
            size='small'
            type='text'
            className='mr-auto text-[10px]'
            onClick={onPreviousArtifact}
          />
        )}

        <div className='scale-90'>
          <div className='text-[10px] mr-1 text-center mb-1'>dozing-oyster</div>

          <Artifact artifact={selectedArtifact} />
        </div>

        {totalArtifacts > 1 && (
          <Button
            icon={<FontAwesomeIcon icon={faChevronRight} />}
            size='small'
            type='text'
            className='ml-auto text-[10px]'
            onClick={onNextArtifact}
          />
        )}
      </Flex>
    </>
  );
};

const Artifact = (props: { artifact: string | GitCommit | Chart | Image }) => {
  if (typeof props.artifact === 'string') {
    return (
      <Typography.Text type='secondary' className='text-xs'>
        Empty Freight
      </Typography.Text>
    );
  }

  const source = (
    <span className='text-[10px] ml-1'>{humanComprehendableArtifact(props.artifact.repoURL)}</span>
  );

  if (props.artifact?.$typeName === 'github.com.akuity.kargo.api.v1alpha1.GitCommit') {
    const url = getGitCommitURL(props.artifact.repoURL, props.artifact.id);

    let TagComponent = (
      <Tag title={props.artifact.repoURL} bordered={false} color='geekblue'>
        <Flex justify='center' align='center'>
          <div>
            {props.artifact.id.slice(0, 7)}

            {!!url && (
              <FontAwesomeIcon icon={faExternalLink} className='text-blue-600 text-[8px] ml-1' />
            )}
          </div>

          {source}
        </Flex>
      </Tag>
    );

    if (url) {
      TagComponent = (
        <Link href={url} target='_blank' onClick={(e) => e.stopPropagation()}>
          {TagComponent}
        </Link>
      );
    }

    return (
      <Flex vertical gap={2}>
        {TagComponent}
        <Typography.Text
          className='text-[10px] text-center'
          type='secondary'
          title={`${props.artifact.author}: ${props.artifact.message}`}
        >
          <FontAwesomeIcon icon={faCodeCommit} className='mr-1' />
          {props.artifact.message?.slice(0, 35)}
          {props.artifact?.message?.length > 35 ? '...' : ''}
        </Typography.Text>
      </Flex>
    );
  }

  if (props.artifact?.$typeName === 'github.com.akuity.kargo.api.v1alpha1.Chart') {
    return (
      <Tag
        title={`${props.artifact.repoURL}:${props.artifact.version}`}
        bordered={false}
        color='geekblue'
      >
        {shortVersion(props.artifact.version)}

        {source}
      </Tag>
    );
  }

  let imageSourceFromOci = '';
  let imageBuiltDate = '';

  if (props.artifact.annotations) {
    imageSourceFromOci = getImageSource(props.artifact.annotations);
    imageBuiltDate = getImageBuiltDate(props.artifact.annotations);
  }

  let TagComponent = (
    <Tag
      title={`${props.artifact.repoURL}:${props.artifact.tag}`}
      bordered={false}
      color='geekblue'
    >
      <Flex justify='center'>
        <div className='text-center'>
          {shortVersion(props.artifact?.tag)}

          {!!imageSourceFromOci && (
            <FontAwesomeIcon icon={faExternalLink} className='text-blue-600 ml-1 text-[8px]' />
          )}
        </div>
        {source}
      </Flex>
    </Tag>
  );

  if (imageSourceFromOci) {
    TagComponent = (
      <Link href={imageSourceFromOci} target='_blank' onClick={(e) => e.stopPropagation()}>
        {TagComponent}
      </Link>
    );
  }

  if (imageBuiltDate) {
    TagComponent = (
      <Flex vertical gap={8}>
        {TagComponent}
        <Typography.Text className='text-[10px] text-center' type='secondary'>
          {imageBuiltDate}
        </Typography.Text>
      </Flex>
    );
  }

  return TagComponent;
};

const useStageHeaderStyle = (stage: Stage): CSSProperties => {
  if (!useIsColorsUsed()) {
    return {};
  }

  let stageColor = parseColorAnnotation(stage);
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

const useIsStageAutoPromotionMode = () => {
  return true;
};

const getStagePhase = (stage: Stage) => stage?.status?.phase as StagePhase;

const isStageControlFlow = (stage: Stage) =>
  (stage?.spec?.promotionTemplate?.spec?.steps?.length || 0) <= 0;

const getStageHealth = (stage: Stage) => stage?.status?.health;

const useIsColorsUsed = () => {
  return false;
};

const getLastPromotion = (stage: Stage) => stage?.status?.lastPromotion?.finishedAt;
