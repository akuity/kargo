import { faArrowRightToBracket, faTimes } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex } from 'antd';
import { useContext } from 'react';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ColorContext } from '@ui/context/colors';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { SmallLabel } from '../common/small-label';
import { StageTag } from '../common/stage-tag';
import { CurrentFreightItem } from '../common/utils';
import { FreightArtifactList } from '../project/pipelines/freight/freight-artifact-list';
import { shortVersion } from '../project/pipelines/freight/short-version-utils';

export const RequestedFreight = ({
  projectName,
  requestedFreight,
  onDelete,
  className,
  itemStyle,
  hideTitle,
  currentFreight
}: {
  projectName?: string;
  requestedFreight?: {
    origin?: { kind?: string; name?: string };
    sources?: { direct?: boolean; stages?: string[] };
  }[];
  onDelete?: (index: number) => void;
  className?: string;
  itemStyle?: React.CSSProperties;
  hideTitle?: boolean;
  currentFreight?: Record<string, CurrentFreightItem>;
}) => {
  const { stageColorMap } = useContext(ColorContext);

  const uniqueUpstreamStages = new Set<string>();

  for (const freight of requestedFreight || []) {
    for (const stage of freight.sources?.stages || []) {
      uniqueUpstreamStages.add(stage);
    }
  }

  if (!requestedFreight || requestedFreight.length === 0) {
    return null;
  }

  return (
    <div className={className}>
      {!hideTitle && <h3>Requested Freight</h3>}

      <div className='flex gap-5 flex-wrap'>
        {requestedFreight?.map((freight, i) => {
          const current = currentFreight?.[`${freight.origin?.kind}/${freight.origin?.name}`];
          const currentFreightName = current?.reference?.name || '';

          return (
            <div
              key={i}
              className='bg-gray-50 rounded-md p-3 border-2 border-solid border-gray-200'
              style={itemStyle}
            >
              <Flex gap={16} align='stretch'>
                <div className='flex-1 min-w-0'>
                  <Flex>
                    <div>
                      <SmallLabel className='mb-1'>
                        {(freight.origin?.kind || 'Unknown').toUpperCase()}
                      </SmallLabel>

                      <div className='mb-3 font-semibold'>{freight.origin?.name}</div>
                    </div>
                    {onDelete && (
                      <div className='ml-auto cursor-pointer'>
                        <FontAwesomeIcon icon={faTimes} onClick={() => onDelete(i)} />
                      </div>
                    )}
                  </Flex>

                  <SmallLabel className='mb-1'>SOURCE</SmallLabel>
                  <Flex gap={6}>
                    {freight.sources?.direct && (
                      <Link
                        to={generatePath(paths.warehouse, {
                          name: projectName,
                          warehouseName: freight.origin?.name
                        })}
                      >
                        <Flex
                          align='center'
                          justify='center'
                          className='bg-gray-600 text-white py-1 px-2 rounded font-semibold cursor-pointer'
                        >
                          <FontAwesomeIcon icon={faArrowRightToBracket} className='mr-2' />
                          DIRECT
                        </Flex>
                      </Link>
                    )}
                    {freight.sources?.stages?.map((stage) => (
                      <Link
                        key={stage}
                        to={generatePath(paths.stage, { name: projectName, stageName: stage })}
                      >
                        <StageTag
                          stage={{ metadata: { name: stage } } as Stage}
                          projectName={projectName || ''}
                          stageColorMap={stageColorMap}
                        />
                      </Link>
                    ))}
                  </Flex>
                </div>

                {current && (
                  <div className='w-48 shrink-0 border-0 border-l border-solid border-gray-200 pl-4'>
                    <SmallLabel className='mb-1'>CURRENT FREIGHT</SmallLabel>
                    <Link
                      to={generatePath(paths.freight, {
                        name: projectName,
                        freightName: currentFreightName
                      })}
                      className='font-semibold truncate block mb-3'
                    >
                      {current.alias || shortVersion(currentFreightName, 12)}
                    </Link>
                    <SmallLabel className='mb-1'>ARTIFACTS</SmallLabel>
                    <Flex
                      wrap
                      gap={4}
                      align='center'
                      className='[&_.ant-tag]:m-0 [&_.ant-tag]:max-w-full [&_.ant-tag]:truncate [&>a]:flex [&>a]:items-center [&_a_.ant-tag]:cursor-pointer'
                    >
                      <FreightArtifactList freight={current.reference} />
                    </Flex>
                  </div>
                )}
              </Flex>
            </div>
          );
        })}
      </div>
    </div>
  );
};
