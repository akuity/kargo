import { Divider, Drawer, Tabs, Typography } from 'antd';
import moment from 'moment';
import { useMemo, useState } from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { Stage, VerificationInfo } from '@ui/gen/v1alpha1/generated_pb';

import { Description } from '../common/description';
import { ManifestPreview } from '../common/manifest-preview';
import { useImages } from '../project/pipelines/utils/useImages';

import { PrLinks } from './pr-links';
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
    return (
      (stage.status?.freightHistory
        ? stage.status?.freightHistory
        : (stage.status?.history as { verificationHistory: VerificationInfo[] }[])) || []
    )
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

  const repoUrls = useMemo(() => {
    return (stage.spec?.promotionMechanisms?.gitRepoUpdates || []).map((g) => g.repoURL || '');
  }, [stage]);

  return (
    <Drawer open={!!stageName} onClose={onClose} width={'80%'} closable={false}>
      {stage && (
        <div className='flex flex-col h-full'>
          <div className='flex items-center justify-between'>
            <div className='flex gap-1 items-start'>
              <HealthStatusIcon
                health={stage.status?.health}
                style={{ marginRight: '10px', marginTop: '10px' }}
              />
              <div>
                <Typography.Title level={1} style={{ margin: 0 }}>
                  {stage.metadata?.name}
                </Typography.Title>
                <Typography.Text type='secondary'>{projectName}</Typography.Text>
                <Description item={stage} loading={false} className='mt-2' />
              </div>
            </div>
            <div className='ml-auto mr-4'>
              <PrLinks
                repoUrls={repoUrls}
                metadata={stage.status?.lastPromotion?.status?.metadata}
              />
            </div>
            <StageActions stage={stage} verificationRunning={isVerificationRunning} />
          </div>
          <Divider style={{ marginTop: '1em' }} />

          <div className='flex flex-col gap-8 flex-1'>
            <RequestedFreight stage={stage} projectName={projectName} />
            <Tabs
              className='flex-1'
              defaultActiveKey='1'
              style={{ minHeight: '500px' }}
              items={[
                {
                  key: '1',
                  label: 'Promotions',
                  children: <Promotions repoUrls={repoUrls} />
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
                  children: <ManifestPreview object={stage} />
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
