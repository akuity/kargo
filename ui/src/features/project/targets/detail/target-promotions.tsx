import { Flex, Modal, Table, Tooltip, Typography } from 'antd';
import { ColumnsType } from 'antd/es/table';
import { formatDistanceToNow } from 'date-fns';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { PromotionStatusIcon } from '@ui/features/common/promotion-status/promotion-status-icon';
import { useListTargetPromotions } from '@ui/gen/api/v2/core/core';
import { TargetPromotion } from '@ui/gen/api/v2/models';
import { parseDate } from '@ui/utils/dates';

import { FreightLink } from '../shared/freight-link';
import { useFreightLabel } from '../shared/use-freight-label';

const pagination = { pageSize: 10, hideOnSinglePage: true, showSizeChanger: false };

/**
 * TargetPromotions lists the Promotions to this Target, newest first, from
 * Kargo's record of every Promotion, so those Kubernetes has since garbage
 * collected are listed too. It lives behind a button rather than on the row:
 * the history is fetched only when someone asks for it.
 */
const TargetPromotions = ({ project, target }: { project: string; target: string }) => {
  const query = useListTargetPromotions(project, target);
  const freightLabel = useFreightLabel(project);
  const promotions = query.data?.data?.items || [];

  const columns: ColumnsType<TargetPromotion> = [
    {
      title: 'Phase',
      key: 'phase',
      width: 130,
      render: (_, promotion) => (
        <Flex gap={8} align='center'>
          <PromotionStatusIcon status={{ phase: promotion.phase }} />
          <span className='text-xs'>{promotion.phase || 'Pending'}</span>
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
            promotionId: promotion.name || ''
          })}
        >
          {promotion.name}
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
          to={generatePath(paths.stage, { name: project, stageName: promotion.stage || '' })}
        >
          {promotion.stage}
        </Link>
      )
    },
    {
      title: 'Freight',
      key: 'freight',
      width: 140,
      render: (_, promotion) => {
        const freight = promotion.freight || '';
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
        const finished = parseDate(promotion.finishedAt);
        const created = parseDate(promotion.createdAt);
        const when = finished || created;
        return (
          <Tooltip title={when?.toLocaleString()}>
            <Typography.Text type='secondary' className='text-xs'>
              {when
                ? `${finished ? 'finished' : 'created'} ${formatDistanceToNow(when, { addSuffix: true })}`
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
      // A name is reused when a Promotion is recreated; its creation is not.
      rowKey={(promotion) => `${promotion.name}@${promotion.createdAt}`}
      locale={{ emptyText: 'No Promotion has named this Target' }}
    />
  );
};

/**
 * TargetPromotionsModal wraps that list for the modal stack. It is handed only
 * the Project and the Target's name, so the list fetches its own Promotions.
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
