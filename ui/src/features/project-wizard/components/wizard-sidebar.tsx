import { faCheck, faLock } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { theme } from 'antd';
import classNames from 'classnames';

import { STEP_META, isStepComplete, isStepLocked, stepIndex } from '../step-meta';
import { StepId, WizardState } from '../types';

type WizardSidebarProps = {
  current: StepId;
  state: WizardState;
  onJump: (id: StepId) => void;
};

export const WizardSidebar = ({ current, state, onJump }: WizardSidebarProps) => {
  const { token } = theme.useToken();
  const progress = Math.round((stepIndex(current) / (STEP_META.length - 1)) * 100);

  return (
    <aside className='w-60 2xl:w-72 shrink-0 bg-white p-5 overflow-y-auto'>
      <h2 className='text-base font-semibold text-gray-900 m-0'>Create project</h2>
      <div className='text-xs text-gray-500 mt-1'>
        Set up Kargo resources for a new delivery pipeline.
      </div>
      <div
        className='h-1 rounded mt-4 mb-5 overflow-hidden'
        style={{ backgroundColor: token.colorFillSecondary }}
      >
        <div
          className='h-full rounded transition-all'
          style={{ width: `${Math.max(progress, 6)}%`, backgroundColor: token.colorPrimary }}
        />
      </div>
      <div>
        {STEP_META.map((s) => {
          const isCurrent = s.id === current;
          const locked = isStepLocked(s, state);
          // Completion is not indicated on the step you're currently editing.
          const complete = !locked && !isCurrent && isStepComplete(s.id, state);
          const meta = locked
            ? 'Add stages first'
            : complete
              ? 'Completed'
              : s.required
                ? 'Required'
                : 'Optional';

          return (
            <div
              key={s.id}
              role='button'
              className={classNames('flex items-start gap-3 rounded-md px-2 py-2.5 select-none', {
                'opacity-55 cursor-not-allowed': locked,
                'cursor-pointer hover:bg-gray-50': !locked && !isCurrent
              })}
              style={isCurrent ? { backgroundColor: token.colorFillTertiary } : undefined}
              onClick={() => !locked && onJump(s.id)}
            >
              <div
                className='flex items-center justify-center w-6 h-6 rounded-full border text-xs shrink-0 mt-px'
                style={
                  isCurrent
                    ? {
                        backgroundColor: token.colorPrimary,
                        borderColor: token.colorPrimary,
                        color: token.colorWhite
                      }
                    : complete
                      ? { borderColor: token.colorSuccess, color: token.colorSuccess }
                      : { borderColor: token.colorBorder, color: token.colorTextTertiary }
                }
              >
                {locked ? (
                  <FontAwesomeIcon icon={faLock} size='xs' />
                ) : complete ? (
                  <FontAwesomeIcon icon={faCheck} size='xs' />
                ) : (
                  s.num
                )}
              </div>
              <div className='min-w-0'>
                <div
                  className={classNames(
                    'text-[13px] leading-5',
                    isCurrent ? 'font-semibold text-gray-900' : 'text-gray-700'
                  )}
                >
                  {s.title}
                </div>
                <div
                  className='text-xs'
                  style={{ color: complete ? token.colorSuccess : token.colorTextTertiary }}
                >
                  {meta}
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </aside>
  );
};
