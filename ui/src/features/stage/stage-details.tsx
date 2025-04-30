import { useQuery } from '@connectrpc/connect-query';
import {
  faBarsStaggered,
  faCircleCheck,
  faCircleUp,
  faGear,
  faHistory
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Drawer, Flex, Skeleton, Tabs, Typography } from 'antd';
import moment from 'moment';
import { useEffect, useMemo, useState } from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useExtensionsContext } from '@ui/extensions/extensions-context';
import { Description } from '@ui/features/common/description';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import {
  getConfig,
  getStage
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { RawFormat } from '@ui/gen/api/service/v1alpha1/service_pb';
import { Stage, VerificationInfo } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';
import { decodeRawData } from '@ui/utils/decode-raw-data';

import YamlEditor from '../common/code-editor/yaml-editor-lazy';
import { StageConditionIcon } from '../common/stage-status/stage-condition-icon';

import { Promotions } from './promotions';
import { RequestedFreight } from './requested-freight';
import { StageActions } from './stage-actions';
import { FreightHistory } from './tabs/freight-history/freight-history';
import { StageSettings } from './tabs/settings/stage-settings';
import { useImages } from './use-images';
import { Verifications } from './verifications';

enum TabsTypes {
  PROMOTION = 'Promotion',
  VERIFICATIONS = 'Verification',
  LIVE_MANIFEST = 'Live Manifest',
  FREIGHT_HISTORY = 'Freight History',
  SETTINGS = 'Settings'
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

  const { data: config } = useQuery(getConfig);
  const shardKey = stage?.metadata?.labels['kargo.akuity.io/shard'] || '';
  const argocdShard = config?.argocdShards?.[shardKey];

  const { stageTabs } = useExtensionsContext();

  const stageConditions = useMemo(() => stage.status?.conditions || [], [stage.status?.conditions]);

  return (
    <Drawer
      open={!!stageName}
      onClose={onClose}
      width='80%'
      title={
        <Flex justify='space-between' className='font-normal'>
          <div>
            <Flex gap={16} align='center'>
              <Typography.Title level={2} style={{ margin: 0 }}>
                {stage.metadata?.name}
              </Typography.Title>
              <Flex gap={4}>
                <StageConditionIcon conditions={stageConditions} />
                {!!stage.status?.health && <HealthStatusIcon health={stage.status?.health} />}
              </Flex>
            </Flex>
            <div className='-mt-1'>
              <Typography.Text type='secondary'>{projectName}</Typography.Text>
              <Description item={stage} loading={false} className='mt-2' />
            </div>
          </div>
          <StageActions
            stage={stage}
            verificationRunning={isVerificationRunning}
            argocdShard={argocdShard}
          />
        </Flex>
      }
    >
      {stage && (
        <div className='flex flex-col h-full'>
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
                  icon: <FontAwesomeIcon icon={faCircleUp} />,
                  children: <Promotions argocdShard={argocdShard} />
                },
                {
                  key: TabsTypes.VERIFICATIONS,
                  label: 'Verifications',
                  icon: <FontAwesomeIcon icon={faCircleCheck} />,
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
                  icon: <FontAwesomeIcon icon={faBarsStaggered} />,
                  className: 'h-full pb-2',
                  children: rawStageYamlQuery.isLoading ? (
                    <Skeleton />
                  ) : (
                    <YamlEditor
                      value={rawStageYaml}
                      height='700px'
                      isHideManagedFieldsDisplayed
                      disabled
                    />
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
          </div>
        </div>
      )}
    </Drawer>
  );
};

export default StageDetails;
