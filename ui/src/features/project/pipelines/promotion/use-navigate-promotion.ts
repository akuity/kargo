import { useCallback } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';

import { createdPromotionState } from './created-promotion';

export const useNavigatePromotion = () => {
  const navigate = useNavigate();

  return useCallback(
    (project: string | undefined, promotion: string | undefined) =>
      navigate(generatePath(paths.promotion, { name: project, promotionId: promotion }), {
        state: createdPromotionState(promotion)
      }),
    [navigate]
  );
};
