import { useQuery } from '@connectrpc/connect-query';
import { faChevronDown, faExternalLink } from '@fortawesome/free-solid-svg-icons';
import { faGear, faHistory } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Divider, Drawer, Skeleton, Space, Tabs, Typography } from 'antd';
import Dropdown from 'antd/es/dropdown/dropdown';
import moment from 'moment';
import { useEffect, useMemo, useState } from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useExtensionsContext } from '@ui/extensions/extensions-context';
import { Description } from '@ui/features/common/description';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { StagePhaseIcon } from '@ui/features/common/stage-phase/stage-phase-icon';
import { StagePhase } from '@ui/features/common/stage-phase/utils';
import { useImages } from '@ui/features/project/pipelines/utils/useImages';
import {
  getConfig,
  getStage
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { RawFormat } from '@ui/gen/api/service/v1alpha1/service_pb';
import { Stage, VerificationInfo } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';
import { decodeRawData } from '@ui/utils/decode-raw-data';

import YamlEditor from '../common/code-editor/yaml-editor-lazy';

import { FreightHistory } from './freight-history';
import { Promotions } from './promotions';
import { RequestedFreight } from './requested-freight';
import { StageActions } from './stage-actions';
import { Verifications } from './verifications';

enum TabsTypes {
  PROMOTION = 'Promotion',
  VERIFICATIONS = 'Verification',
  LIVE_MANIFEST = 'Live Manifest'
}

export const StageDetails = ({ stage }: { stage: Stage }) => {
  const { name: projectName, stageName } = useParams();
  const navigate = useNavigate();

  const images = useImages([stage]);

  const onClose = () => navigate(generatePath(paths.project, { name: projectName }));
  const [isVerificationRunning, setIsVerificationRunning] = useState(false);

  const verifications = useMemo(() => {
    setIsVerificationRunning(false);
    return (stage.status?.freightHistory || [])
      .flatMap((freight) =>
        freight.verificationHistory.map((verification) => {
          if (verification.phase === 'Running' || verification.phase === 'Pending') {
            setIsVerificationRunning(true);
          }
          return {
            ...verification,
            freight
          } as VerificationInfo;
        })
      )
      .sort((a, b) => moment(timestampDate(b.startTime)).diff(moment(timestampDate(a.startTime))));
  }, [stage]);

  const { data: config } = useQuery(getConfig);

  const rawStageYamlQuery = useQuery(getStage, {
    project: projectName,
    name: stage?.metadata?.name,
    format: RawFormat.YAML
  });

  const [activeTab, setActiveTab] = useState(TabsTypes.PROMOTION);

  useEffect(() => {
    if (activeTab === TabsTypes.LIVE_MANIFEST) {
      rawStageYamlQuery.refetch();
    }
  }, [stage, activeTab]);

  const rawStageYaml = useMemo(
    () => decodeRawData(rawStageYamlQuery.data),
    [rawStageYamlQuery.data]
  );

  const shardKey = stage?.metadata?.labels['kargo.akuity.io/shard'] || '';
  const argocdShard = config?.argocdShards?.[shardKey];

  const argocdLinks = useMemo(() => {
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
  const { stageTabs } = useExtensionsContext();

  return (
    <Drawer open={!!stageName} onClose={onClose} width={'80%'} closable={false}>
      {stage && (
        <div className='flex flex-col h-full'>
          <div className='flex items-center justify-between'>
            <div className='flex gap-1 items-center'>
              <StagePhaseIcon className='mr-2' phase={stage.status?.phase as StagePhase} />
              {!!stage.status?.health && (
                <HealthStatusIcon health={stage.status?.health} style={{ marginRight: '10px' }} />
              )}
              <div>
                <Typography.Title level={1} style={{ margin: 0 }}>
                  {stage.metadata?.name}
                </Typography.Title>
                <Typography.Text type='secondary'>{projectName}</Typography.Text>
                <Description item={stage} loading={false} className='mt-2' />
              </div>
            </div>
            {argocdLinks?.length > 0 ? (
              <div className='ml-auto mr-5 text-base flex gap-2 items-center'>
                {argocdLinks?.length === 1 && (
                  <a
                    target='_blank'
                    href={argocdLinks[0]}
                    className='ml-auto mr-5 text-base flex gap-2 items-center'
                  >
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
                ArgoCD
                <FontAwesomeIcon icon={faExternalLink} className='text-xs' />
              </a>
            ) : null}
            <StageActions stage={stage} verificationRunning={isVerificationRunning} />
          </div>
          <Divider style={{ marginTop: '1em' }} />

          <div className='flex flex-col gap-8 flex-1 pb-10'>
            <RequestedFreight
              requestedFreight={stage?.spec?.requestedFreight || []}
              projectName={projectName}
              itemStyle={{ width: '250px' }}
              className='space-y-5'
            />
            <Tabs
              className='flex-1'
              defaultActiveKey='1'
              style={{ minHeight: 'fit-content' }}
              activeKey={activeTab}
              onChange={(newActiveTab) => setActiveTab(newActiveTab as TabsTypes)}
              items={[
                {
                  key: TabsTypes.PROMOTION,
                  label: 'Promotions',
                  children: <Promotions argocdShard={argocdShard} />
                },
                {
                  key: TabsTypes.VERIFICATIONS,
                  label: 'Verifications',
                  children: (
                    <Verifications
                      verifications={verifications}
                      images={Array.from(images.keys())}
                    />
                  )
                },
                {
                  key: TabsTypes.LIVE_MANIFEST,
                  label: 'Live Manifest',
                  className: 'h-full pb-2',
                  children: rawStageYamlQuery.isLoading ? (
                    <Skeleton />
                  ) : (
                    <YamlEditor value={rawStageYaml} height='700px' isHideManagedFieldsDisplayed />
                  )
                },
                {
                  key: TabsTypes.FREIGHT_HISTORY,
                  label: 'Freight History',
                  icon: <FontAwesomeIcon icon={faHistory} />,
                  children: (
                    <FreightHistory
                      requestedFreights={stage?.spec?.requestedFreight || []}
                      freightHistory={stage?.status?.freightHistory}
                      currentActiveFreight={stage?.status?.lastPromotion?.freight?.name}
                      projectName={projectName || ''}
                    />
                  )
                },
                ...stageTabs.map((data, index) => ({
                  children: <data.component stage={stage} />,
                  key: String(data.label + index),
                  label: data.label,
                  icon: data.icon
                })),
                {
                  key: TabsTypes.SETTINGS,
                  label: 'Settings',
                  icon: <FontAwesomeIcon icon={faGear} />,
                  children: <StageSettings />
                }
              ]}
            />

            <FreightHistory
              requestedFreights={stage?.spec?.requestedFreight || []}
              freightHistory={stage?.status?.freightHistory}
              currentActiveFreight={stage?.status?.lastPromotion?.freight?.name}
              projectName={projectName || ''}
            />
          </div>
        </div>
      )}
    </Drawer>
  );
};

export default StageDetails;
