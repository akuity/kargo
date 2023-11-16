import { faChevronDown, faPen, faRefresh, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useQuery } from '@tanstack/react-query';
import { Button, Dropdown, Space } from 'antd';
import React from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import {
  deleteStage,
  queryFreight,
  getConfig,
  refreshStage
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Stage } from '@ui/gen/v1alpha1/types_pb';

import { ButtonIcon } from '../common';
import { useConfirmModal } from '../common/confirm-modal/use-confirm-modal';
import { useModal } from '../common/modal/use-modal';

import { EditStageModal } from './edit-stage-modal';

export const StageActions = ({ stage }: { stage: Stage }) => {
  const { name: projectName, stageName } = useParams();
  const navigate = useNavigate();
  const confirm = useConfirmModal();
  const queryClient = useQueryClient();
  const [shouldRefetchFreights, setShouldRefetchFreights] = React.useState(false);

  const { mutate, isPending: isLoadingDelete } = useMutation(deleteStage.useMutation());
  const { mutate: refresh, isPending: isRefreshLoading } = useMutation(refreshStage.useMutation());

  const onClose = () => navigate(generatePath(paths.project, { name: projectName }));

  const onDelete = () => {
    confirm({
      onOk: () => {
        mutate({ name: stage.metadata?.name, project: projectName });
        onClose();
      },
      title: 'Are you sure you want to delete Stage?'
    });
  };

  const onRefresh = () => refresh({ name: stageName, project: projectName });

  const { show: showEditStageModal } = useModal((p) =>
    stageName && projectName ? (
      <EditStageModal {...p} stageName={stageName} projectName={projectName} />
    ) : null
  );

  // Once the Refresh process is done, refetch Freigths list
  React.useEffect(() => {
    if (stage?.metadata?.annotations['kargo.akuity.io/refresh']) {
      setShouldRefetchFreights(true);
    }

    if (!stage?.metadata?.annotations['kargo.akuity.io/refresh'] && shouldRefetchFreights) {
      queryClient.invalidateQueries({ queryKey: queryFreight.getPartialQueryKey() });
      setShouldRefetchFreights(false);
    }
  }, [stage, shouldRefetchFreights]);

  const { data: config } = useQuery(getConfig.useQuery());
  const argoCDAppsLinks = React.useMemo(() => {
    const shardKey = stage?.metadata?.annotations['kargo.akuity.io/shard'] || '';
    const shard = config?.argocdShards?.[shardKey];

    if (!shard || !stage.spec?.promotionMechanisms?.argocdAppUpdates.length) {
      return [];
    }

    return stage.spec?.promotionMechanisms?.argocdAppUpdates.map((argoCD) => ({
      label: argoCD.appName,
      url: `${shard.url}/applications/${shard.namespace}/${argoCD.appName}`
    }));
  }, [config, stage]);

  return (
    <Space size={16}>
      {argoCDAppsLinks.length === 1 && (
        <Button
          type='link'
          onClick={() => window.open(argoCDAppsLinks[0]?.url, '_blank', 'noreferrer')}
          size='small'
        >
          Argo CD
        </Button>
      )}
      {argoCDAppsLinks.length > 1 && (
        <Dropdown
          menu={{
            items: argoCDAppsLinks.map((item, i) => ({
              label: (
                <a href={item?.url} target='_blank' rel='noreferrer'>
                  {item?.label}
                </a>
              ),
              key: i
            }))
          }}
          trigger={['click']}
        >
          <Button type='link' size='small'>
            <Space size={6}>
              Argo CD
              <FontAwesomeIcon icon={faChevronDown} />
            </Space>
          </Button>
        </Dropdown>
      )}
      <Button
        type='default'
        icon={<ButtonIcon icon={faPen} size='1x' />}
        onClick={() => showEditStageModal()}
      >
        Edit
      </Button>
      <Button
        type='default'
        icon={<ButtonIcon icon={faRefresh} size='1x' />}
        onClick={onRefresh}
        loading={isRefreshLoading || !!stage?.metadata?.annotations['kargo.akuity.io/refresh']}
      >
        Refresh
      </Button>
      <Button
        danger
        type='text'
        icon={<ButtonIcon icon={faTrash} size='1x' />}
        onClick={onDelete}
        loading={isLoadingDelete}
        size='small'
      >
        Delete
      </Button>
    </Space>
  );
};
