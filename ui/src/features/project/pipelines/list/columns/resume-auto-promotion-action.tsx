import { faPlay } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button } from 'antd';
import { useMemo, useState } from 'react';

import {
  autoPromotionHoldStateActive,
  getAutoPromotionHoldEntries
} from '@ui/features/project/pipelines/promotion/auto-promotion';
import { ResumeAutoPromotionDrawer } from '@ui/features/project/pipelines/promotion/resume-auto-promotion-drawer';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

export const ResumeAutoPromotionAction = ({ stage }: { stage: Stage }) => {
  const [open, setOpen] = useState(false);
  const hasActiveAutoPromotionHold = useMemo(
    () =>
      getAutoPromotionHoldEntries(stage).some(
        (entry) => entry.hold.state === autoPromotionHoldStateActive
      ),
    [stage]
  );

  if (!hasActiveAutoPromotionHold) {
    return null;
  }

  return (
    <>
      <Button size='small' icon={<FontAwesomeIcon icon={faPlay} />} onClick={() => setOpen(true)}>
        Resume
      </Button>
      <ResumeAutoPromotionDrawer stage={stage} open={open} onClose={() => setOpen(false)} />
    </>
  );
};
