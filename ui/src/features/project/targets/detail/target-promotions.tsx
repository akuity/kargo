import { Flex, Modal, Table, Tooltip, Typography } from 'antd';
import { ColumnsType } from 'antd/es/table';
import { formatDistanceToNow } from 'date-fns';
import { useMemo } from 'react';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { PromotionStatusIcon } from '@ui/features/common/promotion-status/promotion-status-icon';
import { useListPromotions } from '@ui/gen/api/v2/core/core';
import { Promotion } from '@ui/gen/api/v2/models';
import { parseDate } from '@ui/utils/dates';

import { FreightLink } from '../shared/freight-link';
import { useFreightLabel } from '../shared/use-freight-label';

const pagination = { pageSize: 10, hideOnSinglePage: true, showSizeChanger: false };

const startedAt = (promotion: Promotion): number =>
  parseDate(promotion.status?.startedAt)?.getTime() ??
  parseDate(promotion.metadata?.creationTimestamp)?.getTime() ??
  0;

/**
 * TargetPromotions lists every Promotion that named this Target, newest first.
 *
 * A Promotion records the Target it promoted to in its spec, but the API offers
 * no filter for it, so this reads the Project's Promotions and picks out the
 * Target's own. That is why it lives behind a button rather than on the row:
 * the list is fetched only when someone asks for it.
 */
const TargetPromotions = ({ project, target }: { project: string; target: string }) => {
  const query = useListPromotions(project, undefined);
  const freightLabel = useFreightLabel(project);

  const promotions = useMemo(
    () =>
      (query.data?.data?.items || [])
        .filter((promotion) => promotion.spec?.target === target)
        .sort((lhs, rhs) => startedAt(rhs) - startedAt(lhs)),
    [query.data, target]
  );

  const columns: ColumnsType<Promotion> = [
    {
      title: 'Phase',
      key: 'phase',
      width: 130,
      render: (_, promotion) => (
        <Flex gap={8} align='center'>
          <PromotionStatusIcon status={promotion.status} />
          <span className='text-xs'>{promotion.status?.phase || 'Pending'}</span>
        </Flex>
      )
    },
    {
      title: 'Promotion',
      key: 'promotion',
      render: (_, promotion) => (
        <Link
          className='text-xs font-mono'
          to={generatePath(paths.promotion, {
            name: project,
            promotionId: promotion.metadata?.name || ''
          })}
        >
          {promotion.metadata?.name}
        </Link>
      )
    },
    {
      title: 'Stage',
      key: 'stage',
      width: 140,
      render: (_, promotion) => (
        <Link
          className='text-xs'
          to={generatePath(paths.stage, { name: project, stageName: promotion.spec?.stage || '' })}
        >
          {promotion.spec?.stage}
        </Link>
      )
    },
    {
      title: 'Freight',
      key: 'freight',
      width: 140,
      render: (_, promotion) => {
        const freight = promotion.spec?.freight || '';
        return freight ? (
          <FreightLink
            project={project}
            name={freight}
            label={freightLabel(freight)}
            className='text-xs'
          />
        ) : (
          <Typography.Text type='secondary' className='text-xs'>
            none
          </Typography.Text>
        );
      }
    },
    {
      title: 'When',
      key: 'when',
      width: 150,
      render: (_, promotion) => {
        const finished = parseDate(promotion.status?.finishedAt);
        const started = parseDate(promotion.status?.startedAt);
        const when = finished || started;
        return (
          <Tooltip title={when?.toLocaleString()}>
            <Typography.Text type='secondary' className='text-xs'>
              {when
                ? `${finished ? 'finished' : 'started'} ${formatDistanceToNow(when, { addSuffix: true })}`
                : 'not yet'}
            </Typography.Text>
          </Tooltip>
        );
      }
    }
  ];

  return (
    <Table
      dataSource={promotions}
      columns={columns}
      size='small'
      loading={query.isLoading}
      pagination={pagination}
      rowKey={(promotion) => promotion.metadata?.name || ''}
      locale={{ emptyText: 'No Promotion has named this Target' }}
    />
  );
};

/**
 * TargetPromotionsModal wraps that list for the modal stack. It is handed only
 * the Project and the Target's name, so the list fetches its own Promotions and
 * stays live for as long as it is open.
 */
export const TargetPromotionsModal = ({
  project,
  target,
  ...props
}: ModalComponentProps & { project: string; target: string }) => (
  <Modal
    open={props.visible}
    onCancel={props.hide}
    footer={null}
    width='min(1000px, 90vw)'
    title={`Promotions to ${target}`}
  >
    <TargetPromotions project={project} target={target} />
  </Modal>
);
