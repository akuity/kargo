import { faBox, faClock } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useQueries } from '@tanstack/react-query';
import moment from 'moment';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { getAlias } from '@ui/features/common/utils';
import { getLastPromotionRef } from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { getGetFreightQueryOptions, useGetPromotion } from '@ui/gen/api/v2/core/core';
import { FreightReference, Stage } from '@ui/gen/api/v2/models';

export const StagePopover = ({ project, stage }: { project?: string; stage?: Stage }) => {
  const lastPromotion = getLastPromotionRef(stage);

  // A classic Stage's last promotion is a Promotion, which is read to date it
  // by its creation. A target-aware Stage's is a PromotionRequest, for which
  // the Stage's own reference already carries the time it finished.
  const isPromotion = lastPromotion?.kind === 'Promotion';
  const getPromotionQuery = useGetPromotion(project || '', lastPromotion?.name || '', {
    query: { enabled: isPromotion && !!lastPromotion?.name }
  });

  const lastPromotedAt = isPromotion
    ? getPromotionQuery.data?.data?.metadata?.creationTimestamp
    : lastPromotion?.finishedAt;

  const freightData = useQueries({
    queries: Object.values(stage?.status?.freightHistory?.[0]?.items || {}).map(
      (freight: FreightReference) => {
        return getGetFreightQueryOptions(project || '', freight.name || '');
      }
    )
  });

  const _label = ({ children }: { children: string }) => (
    <div className='text-xs font-semibold text-gray-300 mb-1'>{children}</div>
  );

  const navigate = useNavigate();

  return (
    <div>
      <_label>LAST PROMOTED</_label>
      <div className='flex items-center mb-4'>
        <FontAwesomeIcon icon={faClock} className='mr-2' />
        <div>{moment(lastPromotedAt).format('MMM do yyyy HH:mm:ss')}</div>
      </div>
      <_label>CURRENT FREIGHT</_label>
      {Object.values(stage?.status?.freightHistory?.[0]?.items || {}).map((_, i) => (
        <div className='flex items-center mb-2' key={i}>
          <FontAwesomeIcon icon={faBox} className='mr-2' />
          <div>{getAlias(freightData[i]?.data?.data)}</div>
        </div>
      ))}
      <div
        onClick={(e) => {
          e.preventDefault();
          navigate(generatePath(paths.stage, { name: project, stageName: stage?.metadata?.name }));
        }}
        className='underline text-blue-400 font-semibold w-full text-center cursor-pointer'
      >
        DETAILS
      </div>
    </div>
  );
};
