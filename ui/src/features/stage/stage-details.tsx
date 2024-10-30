import { Divider, Drawer, Tabs, Typography } from 'antd';
import moment from 'moment';
import { useMemo, useState } from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { Description } from '@ui/features/common/description';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { ManifestPreview } from '@ui/features/common/manifest-preview';
import { StagePhaseIcon } from '@ui/features/common/stage-phase/stage-phase-icon';
import { StagePhase } from '@ui/features/common/stage-phase/utils';
import { useImages } from '@ui/features/project/pipelines/utils/useImages';
import { Stage, VerificationInfo } from '@ui/gen/v1alpha1/generated_pb';

import { FreightHistory } from './freight-history';
import { Promotions } from './promotions';
import { RequestedFreight } from './requested-freight';
import { StageActions } from './stage-actions';
import { Verifications } from './verifications';

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
            ...verification
          } as VerificationInfo;
        })
      )
      .sort((a, b) => moment(b.startTime?.toDate()).diff(moment(a.startTime?.toDate())));
  }, [stage]);

  return (
    <Drawer open={!!stageName} onClose={onClose} width={'80%'} closable={false}>
      {stage && (
        <div className='flex flex-col h-full'>
          <div className='flex items-center justify-between'>
            <div className='flex gap-1 items-start'>
              <StagePhaseIcon className='mt-2 mr-2' phase={stage.status?.phase as StagePhase} />
              {!!stage.status?.health && (
                <HealthStatusIcon
                  health={stage.status?.health}
                  style={{ marginRight: '10px', marginTop: '10px' }}
                />
              )}
              <div>
                <Typography.Title level={1} style={{ margin: 0 }}>
                  {stage.metadata?.name}
                </Typography.Title>
                <Typography.Text type='secondary'>{projectName}</Typography.Text>
                <Description item={stage} loading={false} className='mt-2' />
              </div>
            </div>
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
              items={[
                {
                  key: '1',
                  label: 'Promotions',
                  children: <Promotions />
                },
                {
                  key: '2',
                  label: 'Verifications',
                  children: (
                    <Verifications
                      verifications={verifications}
                      images={Array.from(images.keys())}
                    />
                  )
                },
                {
                  key: '3',
                  label: 'Live Manifest',
                  className: 'h-full pb-2',
                  children: <ManifestPreview object={stage} height='700px' />
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
