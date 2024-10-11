import { createConnectQueryKey } from '@connectrpc/connect-query';
import { faHistory } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useQueryClient } from '@tanstack/react-query';
import { Empty } from 'antd';
import classNames from 'classnames';
import { generatePath, Link, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import freightTimelineStyles from '@ui/features/freight-timeline/freight-timeline.module.less';
import { queryFreight } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { QueryFreightResponse } from '@ui/gen/service/v1alpha1/service_pb';
import {
  Freight,
  FreightReference,
  FreightRequest,
  StageStatus
} from '@ui/gen/v1alpha1/generated_pb';

import { FreightContents } from '../freight-timeline/freight-contents';
import { FreightItemLabel } from '../freight-timeline/freight-item-label';
import { FreightTimelineWrapper } from '../freight-timeline/freight-timeline-wrapper';

import requestedFreightStyles from './requested-freight.module.less';

export const FreightHistory = ({
  projectName,
  freightHistory,
  requestedFreights,
  className
}: {
  className?: string;
  requestedFreights: FreightRequest[];
  projectName: string;
  // show the freight history thats 1:1 with requested freight
  freightHistory?: StageStatus['freightHistory'];
  // freight hash name which is active at the moment
  // you can get this from lastPromotion in stage status
  // usually last one is active but we have to consider multi-pipeline case
  currentActiveFreight?: string;
}) => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();

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

  return (
    <div className={className}>
      <h3>
        <FontAwesomeIcon icon={faHistory} className='mr-2' />
        Freight History
      </h3>

      {requestedFreights?.map((freight, i) => {
        const freightUniqueIdentifier = `${freight.origin?.kind}/${freight.origin?.name}`;

        const freightReferences = freightHistoryPerWarehouse[freightUniqueIdentifier] || [];

        return (
          <>
            <Link
              className='block'
              style={{ marginBottom: '16px', marginTop: '32px' }}
              to={generatePath(paths.warehouse, {
                name: projectName,
                warehouseName: freight?.origin?.name
              })}
            >
              {freightUniqueIdentifier}
            </Link>
            <div key={i} className='py-5 bg-gray-50'>
              <div className='flex gap-8'>
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
                            idx === 0 && requestedFreightStyles['active-freight-item']
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
            </div>
          </>
        );
      })}
    </div>
  );
};
