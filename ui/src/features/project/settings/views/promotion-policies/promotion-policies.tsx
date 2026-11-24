import { faPlus, faQuestionCircle } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Flex, Popover, Space, Typography } from 'antd';
import { useParams } from 'react-router-dom';

import { useIsAnyExtensionLoaded } from '@ui/extensions/utils';
import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import { useModal } from '@ui/features/common/modal/use-modal';
import { useGetProjectConfig } from '@ui/gen/api/v2/core/core';
import { ApiError } from '@ui/lib/api/custom-fetch';

import { PromotionPoliciesListView } from './promotion-policies-list-view';
import { promotionPolicyLabel } from './promotion-policy-form-utils';
import { PromotionPolicyModal } from './promotion-policy-modal';
import { useSaveProjectPromotionPolicies } from './use-save-promotion-policies';

export const PromotionPolicies = () => {
  const { name = '' } = useParams();

  const getProjectConfigQuery = useGetProjectConfig(name, {
    query: { meta: { silent404: true } }
  });

  const projectConfig = getProjectConfigQuery.data?.data;
  const promotionPolicies = projectConfig?.spec?.promotionPolicies ?? [];
  const loading = getProjectConfigQuery.isLoading;

  // a missing ProjectConfig is expected -- saving creates it. Any other failure
  // leaves the stored spec unknown, and a save would overwrite all of it
  const { error } = getProjectConfigQuery;
  const loadFailed = !!error && !(error instanceof ApiError && error.isNotFound());

  const { mutate: onUpdate } = useSaveProjectPromotionPolicies(name, projectConfig);

  const showAutoRollback = useIsAnyExtensionLoaded();

  const createModal = useModal();
  const editModal = useModal();
  const confirm = useConfirmModal();

  const onCreate = () =>
    createModal.show((p) => (
      <PromotionPolicyModal
        {...p}
        onSubmit={(promotionPolicy) => onUpdate([...promotionPolicies, promotionPolicy])}
      />
    ));

  const onDelete = (index: number) => {
    const promotionPolicy = promotionPolicies[index];

    confirm({
      title: 'Delete Promotion Policy',
      content: (
        <Typography.Text>
          Are you sure you want to delete the promotion policy for{' '}
          <Typography.Text strong>{promotionPolicyLabel(promotionPolicy)}</Typography.Text>?
        </Typography.Text>
      ),
      onOk: () => onUpdate(promotionPolicies.filter((_, i) => i !== index))
    });
  };

  const onEdit = (index: number) =>
    editModal.show((p) => (
      <PromotionPolicyModal
        {...p}
        editing
        promotionPolicy={promotionPolicies[index]}
        onSubmit={(next) =>
          onUpdate(promotionPolicies.map((policy, i) => (i === index ? next : policy)))
        }
      />
    ));

  return (
    <Card
      title={
        <Space size={4}>
          Promotion Policies
          <Popover
            content={
              <Flex vertical gap={6} className='max-w-xs text-xs'>
                <Typography.Text>
                  A policy governs how Freight reaches the Stages its selector matches. Without one,
                  every promotion waits for someone to ask for it.
                </Typography.Text>
                <Typography.Text>
                  <strong>Auto-promotion</strong> lets new Freight move into those Stages on its
                  own, which suits Stages that subscribe to a Warehouse rather than to an upstream
                  Stage.
                </Typography.Text>
                <Typography.Text>
                  Other conditions still apply: a closed promotion window or a held Stage keeps an
                  automatic promotion from running.
                </Typography.Text>
              </Flex>
            }
          >
            <Typography.Text type='secondary'>
              <FontAwesomeIcon icon={faQuestionCircle} size='xs' />
            </Typography.Text>
          </Popover>
        </Space>
      }
      type='inner'
      className='min-h-full'
      extra={
        // the modal captures the policy list as it opens -- until the
        // ProjectConfig has loaded (or is known not to exist) that list is
        // empty, and saving would replace everything already stored
        <Button
          icon={<FontAwesomeIcon icon={faPlus} size='sm' />}
          onClick={onCreate}
          disabled={loading || loadFailed}
        >
          New Policy
        </Button>
      }
    >
      <PromotionPoliciesListView
        promotionPolicies={promotionPolicies}
        loading={loading}
        showAutoRollback={showAutoRollback}
        onEdit={onEdit}
        onDelete={onDelete}
      />
    </Card>
  );
};
