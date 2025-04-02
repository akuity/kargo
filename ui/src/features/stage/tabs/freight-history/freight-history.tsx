import { create } from '@bufbuild/protobuf';
import { useQuery } from '@connectrpc/connect-query';
import { Card, Descriptions, Flex, Space, Table, Tag } from 'antd';
import { useMemo } from 'react';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { queryFreight } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import {
  Freight,
  FreightReference,
  FreightRequest,
  FreightSchema,
  StageStatus
} from '@ui/gen/api/v1alpha1/generated_pb';
import { PlainMessage } from '@ui/utils/connectrpc-utils';

import { LoadingState } from '../../../common';
import { FreightContents } from '../../../freight-timeline/freight-contents';

export const FreightHistory = ({
  projectName,
  freightHistory,
  requestedFreights
}: {
  requestedFreights: FreightRequest[];
  projectName: string;
  // show the freight history thats 1:1 with requested freight
  freightHistory?: StageStatus['freightHistory'];
  // freight hash name which is active at the moment
  // you can get this from lastPromotion in stage status
  // usually last one is active but we have to consider multi-pipeline case
  currentActiveFreight?: string;
}) => {
  const freightQuery = useQuery(queryFreight, { project: projectName });

  const freightMap = useMemo(() => {
    const freightData = freightQuery.data;
    // generate metadata.name -> full freight data (because history doesn't have it all) to show in freight history
    const freightMap: Record<string, Freight> = {};

    for (const freight of freightData?.groups?.['']?.freight || []) {
      const freightId = freight?.metadata?.name;
      if (freightId) {
        freightMap[freightId] = freight;
      }
    }

    return freightMap;
  }, [freightQuery.data]);

  const freightHistoryPerWarehouse = useMemo(() => {
    // to show the history
    const freightHistoryPerWarehouse: Record<
      string /* warehouse eg. Warehouse/w-1 or Warehouse/w-2 */,
      PlainMessage<FreightReference>[]
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

    return freightHistoryPerWarehouse;
  }, [freightHistory]);

  if (freightQuery.isFetching) {
    return <LoadingState />;
  }

  return (
    <Flex vertical gap={16}>
      {requestedFreights?.map((freight, i) => {
        const freightUniqueIdentifier = `${freight.origin?.kind}/${freight.origin?.name}`;

        const freightReferences = freightHistoryPerWarehouse[freightUniqueIdentifier] || [];

        return (
          <Card size='small' key={i}>
            <Descriptions bordered className='max-w-md mb-4' size='small'>
              <Descriptions.Item label={freight.origin?.kind}>
                <Link
                  to={generatePath(paths.warehouse, {
                    name: projectName,
                    warehouseName: freight?.origin?.name
                  })}
                >
                  {freight.origin?.name}
                </Link>
              </Descriptions.Item>
            </Descriptions>

            <Table
              dataSource={freightReferences}
              size='small'
              pagination={{ hideOnSinglePage: true }}
            >
              <Table.Column<PlainMessage<FreightReference>>
                title='ID'
                width={100}
                render={(value, record) =>
                  freightMap[record?.name || '']?.metadata?.name?.substring(0, 7)
                }
              />
              <Table.Column<PlainMessage<FreightReference>>
                title='Alias'
                render={(value, record, index) => (
                  <Space>
                    <Link
                      to={generatePath(paths.freight, {
                        name: projectName,
                        freightName: record.name
                      })}
                    >
                      {freightMap[record?.name || '']?.alias || record.name}
                    </Link>
                    {index === 0 && <Tag color='success'>Active</Tag>}
                  </Space>
                )}
              />
              <Table.Column<PlainMessage<FreightReference>>
                title='Contents'
                render={(value, record) => (
                  <FreightContents
                    horizontal
                    fullContentVisibility
                    highlighted={false}
                    freight={create(FreightSchema, {
                      metadata: {
                        name: record.name
                      },
                      ...record
                    })}
                  />
                )}
              />
            </Table>
          </Card>
        );
      })}
    </Flex>
  );
};
