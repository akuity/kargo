import { useQuery } from '@connectrpc/connect-query';
import { faBox, faClock } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import moment from 'moment';
import { useMemo } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { getAlias } from '@ui/features/common/utils';
import {
  getFreight,
  getPromotion
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Freight, Promotion } from '@ui/gen/v1alpha1/generated_pb';

export const StagePopover = ({
  promotionName,
  project,
  freightName,
  stageName
}: {
  promotionName: string;
  project?: string;
  freightName?: string;
  stageName?: string;
}) => {
  const { data: promotionData } = useQuery(getPromotion, { name: promotionName, project });
  const promotion = useMemo(() => promotionData?.result?.value as Promotion, [promotionData]);
  const { data: freightData } = useQuery(
    getFreight,
    { name: freightName, project },
    { enabled: !!freightName }
  );

  const _label = ({ children }: { children: string }) => (
    <div className='text-xs font-semibold text-neutral-300 mb-1'>{children}</div>
  );

  const navigate = useNavigate();

  return (
    <div>
      <_label>LAST PROMOTED</_label>
      <div className='flex items-center mb-4'>
        <FontAwesomeIcon icon={faClock} className='mr-2' />
        <div>
          {moment(promotion?.metadata?.creationTimestamp?.toDate()).format('MMM do yyyy HH:mm:ss')}
        </div>
      </div>
      <_label>CURRENT FREIGHT</_label>
      <div className='flex items-center mb-2'>
        <FontAwesomeIcon icon={faBox} className='mr-2' />
        <div>{getAlias(freightData?.result?.value as Freight)}</div>
      </div>
      <div
        onClick={(e) => {
          e.preventDefault();
          navigate(generatePath(paths.stage, { name: project, stageName }));
        }}
        className='underline text-blue-400 font-semibold w-full text-center cursor-pointer'
      >
        DETAILS
      </div>
    </div>
  );
};
