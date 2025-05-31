import { create } from '@bufbuild/protobuf';
import { faTruck } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Descriptions, Flex, Select, Space, Table, Tag, Typography } from 'antd';
import { formatDistance } from 'date-fns';
import { useMemo } from 'react';
import { useState } from 'react';
import { useEffect } from 'react';
import { generatePath, Link, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import {
  FreightReference,
  FreightRequest,
  FreightSchema,
  StageStatus
} from '@ui/gen/api/v1alpha1/generated_pb';
import { PlainMessage, timestampDate } from '@ui/utils/connectrpc-utils';

import { FreightContents } from '../../../freight-timeline/freight-contents';

import { useGetFreightMap } from './use-get-freight-map';
import { usePromotionsByFreightCollection } from './use-promotions-by-freight-collection';

export const FreightHistory = ({
  projectName,
  freightHistory,
  requestedFreights,
  stageName
}: {
  requestedFreights: FreightRequest[];
  projectName: string;
  stageName: string;
  // show the freight history thats 1:1 with requested freight
  freightHistory?: StageStatus['freightHistory'];
  // freight hash name which is active at the moment
  // you can get this from lastPromotion in stage status
  // usually last one is active but we have to consider multi-pipeline case
  currentActiveFreight?: string;
}) => {
  const navigate = useNavigate();
  const promotionsByFreightCollection = usePromotionsByFreightCollection({
    project: projectName,
    stage: stageName
  });

  const [selectedRequestedFreight, setSelectedRequestedFreight] = useState<FreightRequest>();

  const freightMap = useGetFreightMap(projectName);

  const freightHistoryPerWarehouse = useMemo(() => {
    // to show the history
    const freightHistoryPerWarehouse: Record<
      string /* warehouse eg. Warehouse/w-1 or Warehouse/w-2 */,
      PlainMessage<{ id: string } & FreightReference>[]
    > = {};

    for (const freightCollection of freightHistory || []) {
      // key - value
      // warehouse identifier - freight reference
      const items = freightCollection?.items || {};

      for (const [warehouseIdentifier, freightReference] of Object.entries(items)) {
        if (!freightHistoryPerWarehouse[warehouseIdentifier]) {
          freightHistoryPerWarehouse[warehouseIdentifier] = [];
        }

        freightHistoryPerWarehouse[warehouseIdentifier].push({
          id: freightCollection?.id,
          ...freightReference
        });
      }
    }

    return freightHistoryPerWarehouse;
  }, [freightHistory]);

  useEffect(() => {
    if (requestedFreights?.[0]) {
      setSelectedRequestedFreight(requestedFreights[0]);
    }
  }, [requestedFreights]);

  const freightUniqueIdentifier = `${selectedRequestedFreight?.origin?.kind}/${selectedRequestedFreight?.origin?.name}`;
  const freightReferences =
    (freightHistoryPerWarehouse && freightHistoryPerWarehouse[freightUniqueIdentifier]) || [];

  return (
    <Flex vertical gap={16}>
      <Descriptions bordered className='max-w-md' size='small'>
        <Descriptions.Item label={selectedRequestedFreight?.origin?.kind}>
          <Select
            value={freightUniqueIdentifier}
            className='w-full'
            onChange={(value) => {
              const [kind, name] = value.split('/');
              const newRequestedFreight = requestedFreights.find(
                (i) => i.origin?.kind === kind && i.origin?.name === name
              );

              if (newRequestedFreight) {
                setSelectedRequestedFreight(newRequestedFreight);
              }
            }}
            options={requestedFreights?.map((i) => ({
              label: i.origin?.name,
              value: `${i?.origin?.kind}/${i?.origin?.name}`
            }))}
          />
        </Descriptions.Item>
      </Descriptions>

      <Table
        dataSource={freightReferences}
        size='small'
        pagination={{ hideOnSinglePage: true }}
        rowKey={(record, index) => `${record.name}${index}`}
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
          width={240}
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
        <Table.Column<PlainMessage<{ id: string } & FreightReference>>
          render={(_, record) => {
            const freightCreation = timestampDate(
              freightMap[record?.name || '']?.metadata?.creationTimestamp
            );

            let Component;

            if (freightCreation) {
              Component = (
                <Typography.Text type='secondary' className='text-[10px]'>
                  Created: {formatDistance(freightCreation, new Date(), { addSuffix: false })}
                </Typography.Text>
              );
            }

            const promotion = promotionsByFreightCollection[record.id];

            if (promotion) {
              const promotedAt = timestampDate(promotion?.metadata?.creationTimestamp);

              if (promotedAt) {
                Component = (
                  <>
                    {Component}
                    <br />
                    <Typography.Text type='secondary' className='text-[10px]'>
                      Promoted: {formatDistance(promotedAt, new Date(), { addSuffix: false })}
                    </Typography.Text>
                  </>
                );
              }
            }

            return Component;
          }}
        />
        <Table.Column<PlainMessage<FreightReference>>
          render={(_, record, idx) =>
            idx > 0 &&
            freightMap[record?.name] && (
              <Button
                size='small'
                icon={<FontAwesomeIcon icon={faTruck} />}
                onClick={() =>
                  navigate(
                    generatePath(paths.promote, {
                      name: projectName,
                      freight: record.name,
                      stage: stageName
                    })
                  )
                }
              >
                Promote
              </Button>
            )
          }
        />
      </Table>
    </Flex>
  );
};
