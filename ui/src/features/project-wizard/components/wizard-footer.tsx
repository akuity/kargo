import { faChevronLeft, faChevronRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Tooltip } from 'antd';

import { StepId } from '../types';

type WizardFooterProps = {
  current: StepId;
  canContinue: boolean;
  continueDisabledReason?: string;
  continueLabel?: string;
  continueLoading?: boolean;
  backDisabled?: boolean;
  onBack: () => void;
  onContinue: () => void;
};

export const WizardFooter = ({
  current,
  canContinue,
  continueDisabledReason,
  continueLabel,
  continueLoading,
  backDisabled,
  onBack,
  onContinue
}: WizardFooterProps) => {
  const isFirst = current === 'basics';
  const isLast = current === 'review';

  return (
    <div
      // Border spans the full width (incl. under the sidebar); the buttons are
      // inset so Back aligns with the content's left edge, tracking the
      // responsive sidebar: 16.5rem below 2xl (w-60), 19.5rem at 2xl+ (w-72).
      className='sticky bottom-0 flex items-center justify-between bg-white dark:bg-neutral-900 py-4 pr-6 pl-[16.5rem] 2xl:pl-[19.5rem]'
      style={{ borderTop: '2px solid var(--kargo-color-border-secondary, rgba(0,0,0,.05))' }}
    >
      <Button disabled={isFirst || backDisabled} onClick={onBack}>
        <FontAwesomeIcon icon={faChevronLeft} size='sm' />
        Back
      </Button>
      <Tooltip title={canContinue ? undefined : continueDisabledReason}>
        <Button
          type='primary'
          size='large'
          disabled={!canContinue}
          loading={continueLoading}
          onClick={onContinue}
        >
          {continueLabel ?? (isLast ? 'Create project' : 'Save & continue')}
          <FontAwesomeIcon icon={faChevronRight} size='sm' />
        </Button>
      </Tooltip>
    </div>
  );
};
