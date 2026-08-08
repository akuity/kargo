import { Tag } from 'antd';

import { STEP_META, StepMeta } from '../step-meta';

type StepHeadProps = {
  meta: StepMeta;
  locked?: boolean;
};

export const StepHead = ({ meta, locked }: StepHeadProps) => (
  <div className='px-6 pt-6 pb-4'>
    <div className='flex items-center gap-2 text-xs font-medium text-gray-400'>
      Step {meta.num} of {STEP_META.length}
      {meta.required ? (
        <Tag color='orange' className='mr-0'>
          Required
        </Tag>
      ) : (
        <Tag className='mr-0'>Optional</Tag>
      )}
      {locked && (
        <Tag color='gold' className='mr-0'>
          Locked
        </Tag>
      )}
    </div>
    <h1 className='text-2xl font-bold text-gray-900 mt-2 mb-0'>{meta.title}</h1>
    <p className='text-sm text-gray-500 mt-1.5 mb-0 max-w-2xl'>{meta.intro}</p>
  </div>
);
