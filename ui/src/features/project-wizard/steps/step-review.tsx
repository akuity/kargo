import { faCheck, faCircle, faSpinner, faXmark } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Alert, Card, theme } from 'antd';

import { ItemState, ProgressItem } from '../create/create-engine';
import { CreateStatus } from '../create/use-create-project';
import { resourceKey, resourceList } from '../manifest/manifest-builder';
import { WizardState } from '../types';

type StepReviewProps = {
  state: WizardState;
  status: CreateStatus;
  items: ProgressItem[];
};

export const StepReview = ({ state, status, items }: StepReviewProps) => {
  const { token } = theme.useToken();
  const stateIcon: Record<ItemState, { icon: typeof faCheck; color: string; spin?: boolean }> = {
    pending: { icon: faCircle, color: token.colorTextQuaternary },
    running: { icon: faSpinner, color: token.colorPrimary, spin: true },
    done: { icon: faCheck, color: token.colorSuccess },
    error: { icon: faXmark, color: token.colorError }
  };

  // The list is always derived from the current wizard state; run progress (if
  // any) is overlaid per resource by kind/name — never the source of the list.
  const resources = resourceList(state);
  const progressByKey = new Map(items.map((i) => [resourceKey(i), i]));

  return (
    <div className='flex flex-col gap-4'>
      {status === 'error' && (
        <Alert
          type='error'
          showIcon
          message='Creation failed'
          description='One resource could not be created and the rest were halted. Fix the issue (or edit an earlier step) and use Retry — already-created resources are skipped.'
        />
      )}
      <Card title={`Resources (${resources.length})`}>
        {resources.length === 0 ? (
          <div className='text-sm text-gray-400'>
            Nothing to create yet — give your project a name in the first step.
          </div>
        ) : (
          resources.map((r) => {
            const progress = progressByKey.get(resourceKey(r));
            const icon = progress && stateIcon[progress.state];
            return (
              <div key={resourceKey(r)} className='flex items-center gap-2 py-1.5 text-[13px]'>
                {icon ? (
                  <FontAwesomeIcon
                    icon={icon.icon}
                    spin={icon.spin}
                    style={{ color: icon.color }}
                  />
                ) : (
                  <span
                    className='w-1.5 h-1.5 rounded-full'
                    style={{ backgroundColor: token.colorSuccess }}
                  />
                )}
                <span className='font-mono text-gray-700'>
                  {r.kind}/{r.name}
                </span>
                {progress?.message && (
                  <span
                    className='ml-2 text-xs'
                    style={{
                      color: progress.state === 'error' ? token.colorError : token.colorTextTertiary
                    }}
                  >
                    {progress.message}
                  </span>
                )}
              </div>
            );
          })
        )}
      </Card>
    </div>
  );
};
