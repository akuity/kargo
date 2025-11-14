import { useQuery } from '@connectrpc/connect-query';
import { DndContext } from '@dnd-kit/core';
import React, { useEffect, useState } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { queryFreight } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

import { useManualApprovalModal } from '../use-manual-approval-modal';

type Props = React.PropsWithChildren & {
  projectName: string;
};

export const DndPromotionContext = ({ children, projectName }: Props) => {
  const navigate = useNavigate();
  const [stage, setStage] = useState<string>();
  const [freight, setFreight] = useState<string>();

  const query = useQuery(
    queryFreight,
    {
      project: projectName,
      stage
    },
    {
      enabled: !!stage && !!freight
    }
  );

  const showManualApproveModal = useManualApprovalModal();

  useEffect(() => {
    // the query might be triggered from another place, so we need to watch stage and freight too
    if (query.data && stage && freight) {
      const promotionEligibleFreight = query?.data?.groups?.['']?.freight || [];

      const promotionEligible = Boolean(
        promotionEligibleFreight?.find((i) => i?.metadata?.name === freight)
      );

      const navigateToPromotion = () =>
        navigate(
          generatePath(paths.promote, {
            name: projectName,
            freight: freight,
            stage: stage
          })
        );

      if (promotionEligible) {
        navigateToPromotion();
        setStage(undefined);
        setFreight(undefined);
      } else {
        showManualApproveModal({
          freight,
          stage,
          projectName,
          onClose: () => {
            setStage(undefined);
            setFreight(undefined);
          },
          onApprove: () => {
            navigate(
              generatePath(paths.promote, {
                name: projectName,
                freight: freight,
                stage
              })
            );
          }
        });
      }
    }
  }, [query.data, stage, freight]);

  return (
    <DndContext
      autoScroll={false}
      onDragEnd={({ active, over }) => {
        if (
          // make sure that the freight can be promoted to this stage by checking the origin
          over?.data?.current?.requestedFreightNames.includes(active?.data?.current?.originName)
        ) {
          setStage(over.id as string);
          setFreight(active.id as string);
        }
      }}
    >
      {children}
    </DndContext>
  );
};
