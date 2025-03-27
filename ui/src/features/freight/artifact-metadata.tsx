import { faCode, faHammer } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Collapse, Descriptions, Flex, Tag } from 'antd';
import Checkbox from 'antd/es/checkbox/Checkbox';
import { useState } from 'react';

import {
  getImageBuiltDate,
  getImageSource,
  splitOciPrefixedAnnotations
} from '../freight-timeline/open-container-initiative-utils';

import { TableSource } from './flatten-freight-origin-utils';

export const ArtifactMetadata = (props: TableSource) => {
  const [showOnlyOci, setShowOnlyOci] = useState(true);
  if (props.type === 'image') {
    const artifactSource = getImageSource(props?.annotations || {});
    const artifactBuildDate = getImageBuiltDate(props?.annotations || {});
    const { ociPrefixedAnnotations, restAnnotations } = splitOciPrefixedAnnotations(
      props.annotations || {}
    );

    if (
      !artifactSource &&
      !artifactBuildDate &&
      !Object.keys(ociPrefixedAnnotations).length &&
      !Object.keys(restAnnotations).length
    ) {
      return '-';
    }

    return (
      <>
        {(!!artifactSource || !!artifactBuildDate) && (
          <div className='flex gap-4 flex-wrap text-sm'>
            {!!artifactSource && (
              <a href={artifactSource} target='_blank'>
                <FontAwesomeIcon icon={faCode} className='mr-2' /> source code
                <span className='text-[8px] ml-1 font-bold'>OCI</span>
              </a>
            )}

            {!!artifactBuildDate && (
              <div>
                <FontAwesomeIcon icon={faHammer} className='mr-2 text-sm' />
                Built {artifactBuildDate}
                <span className='text-[8px] ml-1 font-bold'>OCI</span>
              </div>
            )}
          </div>
        )}

        {Object.keys(ociPrefixedAnnotations).length > 0 &&
          Object.keys(restAnnotations).length > 0 && (
            <Collapse
              size='small'
              className='mt-2'
              items={[
                {
                  label: (
                    <Flex align='center'>
                      <span className='text-xs mt-1'>Annotations</span>
                      <Checkbox
                        className='ml-auto text-xs'
                        value={showOnlyOci}
                        onChange={(e) => setShowOnlyOci(e.target.checked)}
                        onClick={(e) => e.stopPropagation()}
                      >
                        Only OCI Annotation
                      </Checkbox>
                    </Flex>
                  ),
                  children: (
                    <div className='flex gap-2 flex-wrap'>
                      {Object.entries(ociPrefixedAnnotations)
                        .concat(showOnlyOci ? [] : Object.entries(restAnnotations))
                        .map(([key, value]) => (
                          <Tag key={key}>
                            {key}: {value}
                          </Tag>
                        ))}
                    </div>
                  )
                }
              ]}
            />
          )}
      </>
    );
  }

  if (props.type === 'git') {
    return (
      <Descriptions
        size='small'
        column={1}
        items={[
          {
            label: 'Author',
            children: props.author
          },
          {
            label: 'Branch',
            children: props.branch
          },
          {
            label: 'Committer',
            children: props.committer
          },
          {
            label: 'Message',
            children: props.message
          }
        ]}
      />
    );
  }

  return '-';
};
