import { createConnectQueryKey } from '@connectrpc/connect-query';
import { faArrowRightToBracket, faTimes } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useQueryClient } from '@tanstack/react-query';
import { Empty, Flex } from 'antd';
import classNames from 'classnames';
import { useContext } from 'react';
import { Link, generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ColorContext } from '@ui/context/colors';
import freightTimelineStyles from '@ui/features/freight-timeline/freight-timeline.module.less';
import { queryFreight } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { QueryFreightResponse } from '@ui/gen/service/v1alpha1/service_pb';
import { Freight, FreightReference, Stage, StageStatus } from '@ui/gen/v1alpha1/generated_pb';

import { SmallLabel } from '../common/small-label';
import { StageTag } from '../common/stage-tag';
import { FreightContents } from '../freight-timeline/freight-contents';
import { FreightItemLabel } from '../freight-timeline/freight-item-label';
import { FreightTimelineWrapper } from '../freight-timeline/freight-timeline-wrapper';

import requestedFreightStyles from './requested-freight.module.less';

export const RequestedFreight = ({
  projectName,
  requestedFreight,
  onDelete,
  className,
  itemStyle,
  freightHistory,
  currentActiveFreight
}: {
  projectName?: string;
  requestedFreight?: {
    origin?: { kind?: string; name?: string };
    sources?: { direct?: boolean; stages?: string[] };
  }[];
  onDelete?: (index: number) => void;
  className?: string;
  itemStyle?: React.CSSProperties;
  // show the freight history thats 1:1 with requested freight
  freightHistory?: StageStatus['freightHistory'];
  // freight hash name which is active at the moment
  // you can get this from lastPromotion in stage status
  // usually last one is active but we have to consider multi-pipeline case
  currentActiveFreight?: string;
}) => {
  const queryClient = useQueryClient();
  const { stageColorMap } = useContext(ColorContext);

  const navigate = useNavigate();

  const uniqueUpstreamStages = new Set<string>();

  for (const freight of requestedFreight || []) {
    for (const stage of freight.sources?.stages || []) {
      uniqueUpstreamStages.add(stage);
    }
  }

  // to show the history
  const freightHistoryPerWarehouse: Record<
    string /* warehouse eg. Warehouse/w-1 or Warehouse/w-2 */,
    FreightReference[]
  > = {};

  for (const freightCollection of freightHistory || []) {
    // key - value
    // warehouse identifier - freight reference
    const items = freightCollection?.items || {};

    for (const [warehouseIdentifier, freightReference] of Object.entries(items)) {
      if (!freightHistoryPerWarehouse[warehouseIdentifier]) {
        freightHistoryPerWarehouse[warehouseIdentifier] = [];
      }

      freightHistoryPerWarehouse[warehouseIdentifier].push(freightReference);
    }
  }

  const freightData = queryClient.getQueryData(
    createConnectQueryKey(queryFreight, { project: projectName })
  ) as QueryFreightResponse;

  // generate metadata.name -> full freight data (because history doesn't have it all) to show in freight history
  const freightMap: Record<string, Freight> = {};

  for (const freight of freightData?.groups?.['']?.freight || []) {
    const freightId = freight?.metadata?.name;
    if (freightId) {
      freightMap[freightId] = freight;
    }
  }

  if (!requestedFreight || requestedFreight.length === 0) {
    return null;
  }

  return (
    <>
      <div className='flex items-center gap-8 mb-2'>
        <h3 className='w-2/12'>Requested Freight</h3>
        <h3>Freight History</h3>
      </div>

      <div className={className}>
        {requestedFreight?.map((freight, i) => {
          const freightUniqueIdentifier = `${freight.origin?.kind}/${freight.origin?.name}`;

          const freightReferences = freightHistoryPerWarehouse[freightUniqueIdentifier] || [];

          return (
            <div key={i} className='flex gap-8'>
              {/* REQUESTED FREIGHT BLOCK - TODO(Marvin9): Split into component */}
              <div
                className='bg-gray-50 rounded-md p-3 border-2 border-solid border-gray-200 flex flex-col items-center justify-center'
                style={itemStyle}
              >
                <Flex>
                  <div>
                    <SmallLabel className='mb-1'>
                      {(freight.origin?.kind || 'Unknown').toUpperCase()}
                    </SmallLabel>

                    <div className='text-base mb-3 font-semibold'>{freight.origin?.name}</div>
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

              {/* HISTORY OF REQUESTED FREIGHT - TODO(Marvin9): Splint into component */}
              {freightReferences.length === 0 && (
                <Empty description='No freight history' className='mx-auto' />
              )}
              {freightReferences.length > 0 && (
                <FreightTimelineWrapper containerClassName='py-0'>
                  <div className='flex gap-2 w-full h-full'>
                    {freightReferences.map((freightReference, idx) => (
                      <div
                        key={freightReference.name}
                        className={classNames(
                          freightTimelineStyles.freightItem,
                          'cursor-pointer',
                          idx === 0 &&
                            freightReference?.name === currentActiveFreight &&
                            requestedFreightStyles['active-freight-item']
                        )}
                        onClick={() =>
                          navigate(
                            generatePath(paths.freight, {
                              name: projectName,
                              freightName: freightReference.name
                            })
                          )
                        }
                      >
                        <FreightContents highlighted={false} freight={freightReference} />
                        <div className='text-xs mt-auto'>
                          <FreightItemLabel
                            freight={
                              {
                                ...freightReference,
                                metadata: {
                                  name: freightReference?.name
                                },
                                alias:
                                  freightMap[freightReference?.name || '']?.alias ||
                                  freightReference.name
                              } as Freight
                            }
                          />
                        </div>
                      </div>
                    ))}
                  </div>
                </FreightTimelineWrapper>
              )}
            </div>
          );
        })}
      </div>
    </>
  );
};
