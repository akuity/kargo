import { createConnectQueryKey, useMutation } from '@connectrpc/connect-query';
import {
  faChevronDown,
  faExclamationCircle,
  faExternalLink,
  faRedo,
  faRefresh
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useQueryClient } from '@tanstack/react-query';
import { Button, Dropdown, Space } from 'antd';
import React from 'react';
import { useParams } from 'react-router-dom';

import { transportWithAuth } from '@ui/config/transport';
import {
  abortVerification,
  queryFreight,
  refreshStage,
  reverify
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { ArgoCDShard } from '@ui/gen/api/service/v1alpha1/service_pb';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { currentFreightHasVerification } from '../common/utils';

export const StageActions = ({
  stage,
  verificationRunning,
  argocdShard
}: {
  stage: Stage;
  verificationRunning?: boolean;
  argocdShard?: ArgoCDShard;
}) => {
  const { name: projectName, stageName } = useParams();
  const queryClient = useQueryClient();
  const [shouldRefetchFreights, setShouldRefetchFreights] = React.useState(false);

  const { mutate: refresh, isPending: isRefreshLoading } = useMutation(refreshStage);

  const onRefresh = () => refresh({ name: stageName, project: projectName });

  // Once the Refresh process is done, refetch Freight list
  React.useEffect(() => {
    const refreshRequest = stage?.metadata?.annotations['kargo.akuity.io/refresh'];
    const refreshStatus = stage?.status?.lastHandledRefresh;
    if (refreshRequest !== undefined && refreshRequest !== refreshStatus) {
      setShouldRefetchFreights(true);
    }

    if (refreshRequest === refreshStatus && shouldRefetchFreights) {
      queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({
          schema: queryFreight,
          cardinality: 'finite',
          transport: transportWithAuth
        })
      });
      setShouldRefetchFreights(false);
    }
  }, [stage, shouldRefetchFreights]);

  const { mutate: reverifyStage, isPending } = useMutation(reverify);
  const { mutate: abortVerificationAction } = useMutation(abortVerification);

  const verificationEnabled = stage?.spec?.verification;

  const argocdLinks = React.useMemo(() => {
    const argocdContextKey = 'kargo.akuity.io/argocd-context';

    if (!argocdShard) {
      return [];
    }

    const argocdShardUrl = argocdShard?.url?.endsWith('/')
      ? argocdShard?.url?.slice(0, -1)
      : argocdShard?.url;

    const rawValues = stage.metadata?.annotations?.[argocdContextKey];

    if (!rawValues) {
      return [];
    }

    try {
      const parsedValues = JSON.parse(rawValues) as Array<{
        name: string;
        namespace: string;
      }>;

      return (
        parsedValues?.map(
          (parsedValue) =>
            `${argocdShardUrl}/applications/${parsedValue.namespace}/${parsedValue.name}`
        ) || []
      );
    } catch (e) {
      // deliberately do not crash
      // eslint-disable-next-line no-console
      console.error(e);

      return [];
    }
  }, [argocdShard, stage]);

  return (
    <>
      {argocdLinks?.length > 0 ? (
        <div className='ml-auto mr-5 text-base flex gap-2 items-center'>
          {argocdLinks?.length === 1 && (
            <a
              target='_blank'
              href={argocdLinks[0]}
              className='ml-auto mr-5 text-base flex gap-2 items-center'
            >
              <img src='/argo-logo.svg' alt='Argo' style={{ width: '28px' }} />
              ArgoCD
              <FontAwesomeIcon icon={faExternalLink} className='text-xs' />
            </a>
          )}
          {argocdLinks?.length > 1 && (
            <Dropdown
              menu={{
                items: argocdLinks.map((link, idx) => {
                  const parts = link?.split('/');
                  const name = parts?.[parts.length - 1];
                  const namespace = parts?.[parts.length - 2];
                  return {
                    key: idx,
                    label: (
                      <a target='_blank' href={link}>
                        {namespace} - {name}
                        <FontAwesomeIcon icon={faExternalLink} className='text-xs ml-2' />
                      </a>
                    )
                  };
                })
              }}
            >
              <a onClick={(e) => e.preventDefault()}>
                <Space>
                  <img src='/argo-logo.svg' alt='Argo' style={{ width: '28px' }} />
                  ArgoCD
                  <FontAwesomeIcon icon={faChevronDown} className='text-xs' />
                </Space>
              </a>
            </Dropdown>
          )}
        </div>
      ) : argocdShard?.url ? (
        <a
          target='_blank'
          href={argocdShard?.url}
          className='ml-auto mr-5 text-base flex gap-2 items-center'
        >
          <img src='/argo-logo.svg' alt='Argo' style={{ width: '28px' }} />
          ArgoCD
          <FontAwesomeIcon icon={faExternalLink} className='text-xs' />
        </a>
      ) : null}
      <Space size={16}>
        {currentFreightHasVerification(stage) && (
          <>
            {verificationEnabled && (
              <Button
                icon={<FontAwesomeIcon icon={faRedo} spin={isPending} />}
                disabled={isPending || verificationRunning}
                onClick={() => {
                  reverifyStage({ project: projectName, stage: stageName });
                }}
              >
                Reverify
              </Button>
            )}
            <Button
              type='default'
              disabled={!verificationRunning}
              icon={<FontAwesomeIcon icon={faExclamationCircle} size='1x' />}
              onClick={() => abortVerificationAction({ project: projectName, stage: stageName })}
            >
              Abort Verification
            </Button>
          </>
        )}

        <Button
          type='default'
          icon={<FontAwesomeIcon icon={faRefresh} size='1x' />}
          onClick={onRefresh}
          loading={
            isRefreshLoading ||
            (!!stage?.metadata?.annotations['kargo.akuity.io/refresh'] &&
              stage?.metadata?.annotations?.['kargo.akuity.io/refresh'] !==
                stage?.status?.lastHandledRefresh)
          }
        >
          Refresh
        </Button>
      </Space>
    </>
  );
};
