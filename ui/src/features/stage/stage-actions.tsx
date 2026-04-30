import { createConnectQueryKey, useMutation } from '@connectrpc/connect-query';
import {
  faExclamationCircle,
  faExternalLink,
  faRedo,
  faRefresh
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useQueryClient } from '@tanstack/react-query';
import { Button, Space } from 'antd';
import React from 'react';
import { useParams } from 'react-router-dom';

import { transportWithAuth } from '@ui/config/transport';
import {
  abortVerification,
  queryFreight,
  refreshResource,
  reverify
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { ArgoCDShard } from '@ui/gen/api/service/v1alpha1/service_pb';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { currentFreightHasVerification } from '../common/utils';
import { ArgoCDLink } from '../project/pipelines/nodes/argocd-link';

export const StageActions = ({
  stage,
  verificationRunning
}: {
  stage: Stage;
  verificationRunning?: boolean;
  argocdShard?: ArgoCDShard;
}) => {
  const { name: projectName, stageName } = useParams();
  const queryClient = useQueryClient();
  const [shouldRefetchFreights, setShouldRefetchFreights] = React.useState(false);

  const { mutate: refresh, isPending: isRefreshLoading } = useMutation(refreshResource);

  const refreshResourceTypeStage = 'Stage';
  const onRefresh = () =>
    refresh({
      name: stageName,
      project: projectName,
      resourceType: refreshResourceTypeStage
    });

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

  return (
    <>
      <Space size={16}>
        <ArgoCDLink
          stage={stage}
          externalLinksOnly
          buttonProps={{
            type: 'link',
            iconPosition: 'end',
            icon: <FontAwesomeIcon icon={faExternalLink} size='sm' />
          }}
        >
          <Space size={8}>
            <img src='/argo-logo.svg' alt='ArgoCD' style={{ width: '28px', marginTop: '-2px' }} />
            ArgoCD
          </Space>
        </ArgoCDLink>
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
