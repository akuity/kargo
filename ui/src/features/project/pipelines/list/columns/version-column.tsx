import { faDocker, faGitAlt } from '@fortawesome/free-brands-svg-icons';
import { faAnchor, IconDefinition } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Select, SelectProps, Typography } from 'antd';
import { ColumnType } from 'antd/es/table';
import { useMemo, useState } from 'react';

import { getCurrentFreight } from '@ui/features/common/utils';
import { DictionaryContextType } from '@ui/features/project/pipelines/context/dictionary-context';
import { FreightArtifact } from '@ui/features/project/pipelines/freight/freight-artifact';
import {
  catalogueFreights,
  catalogueFreightVersions,
  filterFreightBySource,
  filterFreightByVersion
} from '@ui/features/project/pipelines/freight/source-catalogue-utils';
import {
  Filter,
  useFilterContext
} from '@ui/features/project/pipelines/list/context/filter-context';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

export const versionColumn = (
  filter: Filter,
  cataloguedFreights: ReturnType<typeof catalogueFreights>,
  cataloguedFreightVersions: ReturnType<typeof catalogueFreightVersions>,
  freightById: DictionaryContextType['freightById']
): ColumnType<Stage> => ({
  title: 'Version',
  render: (_, stage) => {
    const currentFreight = getCurrentFreight(stage);

    // TODO: filter by sources
    const firstFreight = currentFreight[0];

    const totalArtifacts =
      (firstFreight?.commits?.length || 0) +
      (firstFreight?.charts?.length || 0) +
      (firstFreight?.images?.length || 0);

    return (
      <>
        {firstFreight?.commits
          ?.slice(0, 2)
          .map((commit) => <FreightArtifact expand key={commit?.repoURL} artifact={commit} />)}

        {firstFreight?.charts
          ?.slice(0, 2)
          .map((chart) => <FreightArtifact expand key={chart?.repoURL} artifact={chart} />)}

        {firstFreight?.images
          ?.slice(0, 2)
          .map((image) => <FreightArtifact expand key={image?.repoURL} artifact={image} />)}

        {totalArtifacts > 6 && (
          <Typography.Text type='secondary' className='text-[10px]'>
            +{' '}
            {totalArtifacts -
              (firstFreight?.charts?.slice(0, 2)?.length +
                firstFreight?.commits?.slice(0, 2)?.length +
                firstFreight?.images?.slice(0, 2)?.length)}{' '}
            more
          </Typography.Text>
        )}
      </>
    );
  },
  filterDropdown: () => {
    const filters = useFilterContext();

    const [sources, setSources] = useState<string[]>([]);
    const [commits, setCommits] = useState<string[]>([]);
    const [images, setImages] = useState<string[]>([]);
    const [helm, setHelms] = useState<string[]>([]);

    const sourcesDropdownOptions: SelectProps['options'] = useMemo(() => {
      const opts: SelectProps['options'] = [];

      for (const [sourceType, repoURLs] of Object.entries(cataloguedFreights)) {
        let icon: IconDefinition = faGitAlt;

        if (sourceType === 'images') {
          icon = faDocker;
        } else if (sourceType === 'charts') {
          icon = faAnchor;
        }

        for (const repoURL of repoURLs) {
          opts.push({
            value: repoURL,
            label: (
              <div className='w-fit'>
                <FontAwesomeIcon icon={icon} className='mr-2' />
                <span className='text-xs'>{repoURL}</span>
              </div>
            )
          });
        }
      }

      return opts;
    }, [cataloguedFreights]);

    const onApply = () =>
      filters?.onFilter({
        ...filters.filters,
        version: {
          source: sources,
          version: [...commits, ...images, ...helm]
        }
      });

    const onReset = () => filters?.onFilter({ ...filters.filters, version: {} });

    return (
      <Flex style={{ padding: 16 }} vertical gap={16} className='min-w-[500px]'>
        <Flex align='center' gap={8}>
          <label>Source: </label>
          <Select
            className='min-w-[80%] ml-auto'
            placeholder='source'
            options={sourcesDropdownOptions}
            mode='multiple'
            maxTagCount={1}
            value={sources}
            onChange={setSources}
          />
        </Flex>

        <Flex align='center' gap={8}>
          <label>Commit: </label>
          <Select
            className='min-w-[80%] ml-auto'
            placeholder='commit'
            mode='multiple'
            maxTagCount={1}
            options={[...cataloguedFreightVersions.commits].map((commit) => ({
              value: commit,
              label: commit
            }))}
            value={commits}
            onChange={setCommits}
          />
        </Flex>

        <Flex align='center' gap={8}>
          <label>Image: </label>
          <Select
            className='min-w-[80%] ml-auto'
            placeholder='image'
            options={[...cataloguedFreightVersions.images].map((image) => ({
              value: image,
              label: image
            }))}
            mode='multiple'
            maxTagCount={3}
            value={images}
            onChange={setImages}
          />
        </Flex>

        <Flex align='center' gap={8}>
          <label>Helm: </label>
          <Select
            className='min-w-[80%] ml-auto'
            placeholder='helm'
            mode='multiple'
            maxTagCount={4}
            options={[...cataloguedFreightVersions.charts].map((chart) => ({
              value: chart,
              label: chart
            }))}
            value={helm}
            onChange={setHelms}
          />
        </Flex>

        <Button type='primary' size='small' onClick={onApply}>
          Apply
        </Button>

        <Button onClick={onReset} size='small'>
          Reset
        </Button>
      </Flex>
    );
  },
  filteredValue: [...(filter?.version?.source || []), ...(filter?.version?.version || [])],
  onFilter: (_, record) => {
    if (!filter?.version?.source?.length && !filter?.version?.version?.length) {
      return true;
    }

    const currentFreight = getCurrentFreight(record);

    if ((filter?.version?.source?.length || 0) > 0) {
      const filteredFreights = currentFreight.filter((f) =>
        filterFreightBySource(filter.version.source || [])(freightById[f.name])
      );

      if (filteredFreights.length === 0) {
        return false;
      }
    }

    if ((filter?.version?.version?.length || 0) > 0) {
      const filteredFreights = currentFreight.filter((f) =>
        filterFreightByVersion(filter.version.version || [])(freightById[f.name])
      );

      if (filteredFreights.length === 0) {
        return false;
      }
    }

    return true;
  }
});
