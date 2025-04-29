import {
  faChevronLeft,
  faChevronRight,
  faCodeCommit,
  faExternalLink
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Tag, Typography } from 'antd';
import Link from 'antd/es/typography/Link';
import { useEffect, useMemo, useState } from 'react';

import { getCurrentFreight } from '@ui/features/common/utils';
import {
  getGitCommitURL,
  getImageBuiltDate,
  getImageSource
} from '@ui/features/freight-timeline/open-container-initiative-utils';
import { useGetFreightCreation } from '@ui/features/project/pipelines/freight/use-get-freight-creation';
import {
  Chart,
  FreightReference,
  GitCommit,
  Image,
  Stage
} from '@ui/gen/api/v1alpha1/generated_pb';

import './stage-node.less';
import { useDictionaryContext } from '../context/dictionary-context';
import { useFreightTimelineControllerContext } from '../context/freight-timeline-controller-context';
import { humanComprehendableArtifact } from '../freight/artifact-parts-utils';
import { shortVersion } from '../freight/short-version-utils';

import {
  ArtifactTypes,
  selectFirstArtifact,
  selectNextArtifact,
  selectPreviousArtifact
} from './artifact-selector-utils';

export const StageFreight = (props: { stage: Stage }) => {
  const dictionaryContext = useDictionaryContext();
  const freightTimelineControllerContext = useFreightTimelineControllerContext();

  const currentFreight = useMemo(() => getCurrentFreight(props.stage), [props.stage]);

  const warehouses = currentFreight?.map((f) => f.origin?.name);

  const [selectedWarehouse, setSelectedWarehouse] = useState(warehouses?.[0]);

  useEffect(() => {
    if (selectedWarehouse) {
      return;
    }

    setSelectedWarehouse(warehouses?.[0]);
  }, [currentFreight]);

  const defaultToFirstFreight = () =>
    currentFreight?.find((f) => f?.origin?.name === selectedWarehouse) as FreightReference;

  const [selectedFreight, setSelectedFreight] = useState(defaultToFirstFreight);

  useEffect(() => setSelectedFreight(defaultToFirstFreight()), [selectedWarehouse, props.stage]);

  const selectedFreightAlias = useMemo(
    () => dictionaryContext?.freightById?.[selectedFreight?.name]?.alias,
    [selectedFreight]
  );

  const defaultToFirstArtifact = () =>
    // @ts-expect-error FreightReference and Freight are same, at least in this case
    selectFirstArtifact([selectedFreight]) as ArtifactTypes;

  const [selectedArtifact, setSelectedArtifact] = useState(defaultToFirstArtifact());

  useEffect(() => setSelectedArtifact(defaultToFirstArtifact()), [selectedFreight, props.stage]);

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

  const freightCreation = useGetFreightCreation(
    dictionaryContext?.freightById?.[selectedFreight?.name]
  );

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

        <div className='scale-90 flex flex-col items-center'>
          {freightTimelineControllerContext?.preferredFilter?.showAlias && (
            <div className='text-[10px] mr-1 text-center mb-1'>{selectedFreightAlias}</div>
          )}

          <Artifact artifact={selectedArtifact} />

          {freightCreation && (
            <Typography.Text
              type='secondary'
              className='text-[8px] mt-1'
              title={freightCreation?.abs?.toString()}
            >
              {freightCreation?.relative}
            </Typography.Text>
          )}
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
