import { Button, Flex, Modal, Table, Tooltip, Typography } from 'antd';
import { ColumnsType } from 'antd/es/table';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { PromotionStatusIcon } from '@ui/features/common/promotion-status/promotion-status-icon';
import { rolloutPath } from '@ui/features/stage/utils/promotion-rollout';
import { TargetPromotion } from '@ui/gen/api/v2/models';

import { promotionWhen } from '../model/target-details-utils';
import { FreightLink } from '../shared/freight-link';
import { useFreightLabel } from '../shared/use-freight-label';
import { useTargetPromotions } from '../use-target-promotions';

const pagination = { pageSize: 10, hideOnSinglePage: true, showSizeChanger: false };

/**
 * TargetPromotions lists every Promotion to this Target, newest first, from
 * every Stage that has promoted to it.
 *
 * The server keeps the history and reads it a page at a time, so a Target
 * promoted to hundreds of times costs no more to open than one promoted to
 * once; older pages are read when asked for.
 */
const TargetPromotions = ({ project, target }: { project: string; target: string }) => {
  const { promotions, isLoading, hasMore, isLoadingMore, loadMore } = useTargetPromotions(
    project,
    target
  );
  const freightLabel = useFreightLabel(project);

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
          to={generatePath(paths.promotion, { name: project, promotionId: promotion.name || '' })}
        >
          {promotion.name}
        </Link>
      )
    },
    {
      title: 'Stage',
      key: 'stage',
      width: 120,
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
      title: 'Rollout',
      key: 'rollout',
      width: 90,
      render: (_, promotion) =>
        promotion.rollout ? (
          <Link
            className='text-xs'
            to={rolloutPath(project, promotion.stage || '', promotion.rollout)}
          >
            #{promotion.rollout}
          </Link>
        ) : (
          <Typography.Text type='secondary' className='text-xs'>
            none
          </Typography.Text>
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
      width: 170,
      render: (_, promotion) => {
        const { text, date } = promotionWhen(promotion);
        return (
          <Tooltip title={date?.toLocaleString()}>
            <Typography.Text type='secondary' className='text-xs'>
              {text || 'not yet'}
            </Typography.Text>
          </Tooltip>
        );
      }
    }
  ];

  return (
    <Flex vertical gap={8}>
      <Table
        dataSource={promotions}
        columns={columns}
        size='small'
        loading={isLoading}
        pagination={pagination}
        rowKey={(promotion) => promotion.name || ''}
        locale={{ emptyText: 'No Promotion has named this Target' }}
      />
      {hasMore && (
        <Button
          size='small'
          className='self-center'
          loading={isLoadingMore}
          onClick={() => loadMore()}
        >
          Load older Promotions
        </Button>
      )}
    </Flex>
  );
};

/**
 * TargetPromotionsModal wraps that list for the modal stack. It is handed only
 * the Project and the Target's name, so the list reads its own history and
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
