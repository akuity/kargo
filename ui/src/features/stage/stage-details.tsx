import {
  faBarsStaggered,
  faCircleCheck,
  faCircleUp,
  faGear,
  faHistory,
  faPlay
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Drawer, Flex, Skeleton, Tabs, Typography } from 'antd';
import moment from 'moment';
import { useEffect, useMemo, useState } from 'react';
import { generatePath, Link, useNavigate, useParams } from 'react-router-dom';
import { stringify } from 'yaml';

import { SHARD_LABEL_KEY } from '@ui/config/labels';
import { paths } from '@ui/config/paths';
import { useExtensionsContext } from '@ui/extensions/extensions-context';
import { Description } from '@ui/features/common/description';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { useStageControllerStatus } from '@ui/features/common/stage-status/use-stage-controller-status';
import { getCurrentFreightByWarehouse } from '@ui/features/common/utils';
import { getAutoPromotionHoldEntries } from '@ui/features/project/pipelines/promotion/auto-promotion';
import { ResumeAutoPromotionDrawer } from '@ui/features/project/pipelines/promotion/resume-auto-promotion-drawer';
import { useGetStage } from '@ui/gen/api/v2/core/core';
import { Stage } from '@ui/gen/api/v2/models';
import { useGetConfig } from '@ui/gen/api/v2/system/system';

import YamlEditor from '../common/code-editor/yaml-editor-lazy';
import { useModal } from '../common/modal/use-modal';
import { StageConditionIcon } from '../common/stage-status/stage-condition-icon';

import { Promotions } from './promotions';
import { RequestedFreight } from './requested-freight';
import { StageActions } from './stage-actions';
import { FreightHistory } from './tabs/freight-history/freight-history';
import { useGetFreightMap } from './tabs/freight-history/use-get-freight-map';
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

  const freightMap = useGetFreightMap(projectName || '');
  const currentFreight = useMemo(
    () => getCurrentFreightByWarehouse(stage, freightMap),
    [stage, freightMap]
  );

  const onClose = () => navigate(generatePath(paths.project, { name: projectName }));
  const [isVerificationRunning, setIsVerificationRunning] = useState(false);

  const verifications = useMemo(() => {
    setIsVerificationRunning(false);
    return (stage.status?.freightHistory || [])
      .flatMap(
        (freight) =>
          freight.verificationHistory?.map((verification) => {
            if (verification.phase === 'Running' || verification.phase === 'Pending') {
              setIsVerificationRunning(true);
            }
            return {
              ...verification,
              freight
            };
          }) || []
      )
      .sort((a, b) => moment(b?.startTime).diff(moment(a?.startTime)));
  }, [stage]);

  const stageQuery = useGetStage(projectName || '', stage?.metadata?.name || '');

  const [activeTab, setActiveTab] = useState(TabsTypes.PROMOTION);

  useEffect(() => {
    if (activeTab === TabsTypes.LIVE_MANIFEST) {
      stageQuery.refetch();
    }
  }, [stage, activeTab]);

  const rawStageYaml = useMemo(() => stringify(stageQuery.data?.data), [stageQuery.data?.data]);

  const getConfigQuery = useGetConfig();
  const config = getConfigQuery.data?.data;

  const shardKey = stage?.metadata?.labels?.[SHARD_LABEL_KEY] || '';
  const argocdShard = config?.argocdShards?.[shardKey];

  const { stageTabs } = useExtensionsContext();

  const stageConditions = useMemo(() => stage.status?.conditions || [], [stage.status?.conditions]);

  const { controllerName, isControllerDead } = useStageControllerStatus(stage);

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
                <StageConditionIcon
                  conditions={stageConditions}
                  isControllerDead={isControllerDead}
                  controllerName={controllerName}
                />

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
              itemStyle={{ minWidth: '250px', maxWidth: '480px' }}
              className='space-y-5'
              currentFreight={currentFreight}
            />
            <AutoPromotionHolds stage={stage} />
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
                  children: stageQuery.isLoading ? (
                    <Skeleton />
                  ) : (
                    <YamlEditor value={rawStageYaml} height='700px' disabled />
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
                      stageName={stageName || ''}
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

const AutoPromotionHolds = ({ stage }: { stage: Stage }) => {
  const holds = getAutoPromotionHoldEntries(stage);
  const { show } = useModal();

  if (holds.length === 0) {
    return null;
  }
  return (
    <div className='rounded-md border border-solid border-orange-200 bg-orange-50 px-3 py-2'>
      <Flex justify='space-between' align='center' gap={8}>
        <div>
          <Typography.Text strong>Auto-promotion paused</Typography.Text>
          <div className='mt-1 flex flex-col gap-1'>
            {holds.map(({ key, hold }) => (
              <Typography.Text key={key} type='secondary' className='text-sm'>
                {key}
                {hold.freightName && `: ${hold.freightName}`}
                {hold.promotionName && (
                  <>
                    {' via '}
                    <Link
                      to={generatePath(paths.promotion, {
                        name: stage.metadata?.namespace || '',
                        promotionId: hold.promotionName
                      })}
                    >
                      {hold.promotionName}
                    </Link>
                  </>
                )}
                {hold.actor && ` by ${hold.actor}`}
                {hold.createdAt && ` at ${moment(hold.createdAt).format('YYYY-MM-DD HH:mm')}`}
              </Typography.Text>
            ))}
          </div>
        </div>
        <Button
          size='small'
          icon={<FontAwesomeIcon icon={faPlay} />}
          onClick={() =>
            show((p) => (
              <ResumeAutoPromotionDrawer stage={stage} open={p.visible} onClose={p.hide} />
            ))
          }
        >
          Resume
        </Button>
      </Flex>
    </div>
  );
};

export default StageDetails;
