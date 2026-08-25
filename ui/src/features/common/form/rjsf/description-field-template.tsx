import { DescriptionFieldProps } from '@rjsf/utils';

import { useRjsfConfigContext } from './context';

export const DescriptionFieldTemplate = (props: DescriptionFieldProps) => {
  const rjsfConfigContext = useRjsfConfigContext();

  if (!rjsfConfigContext.showDescription) {
    return null;
  }

  return (
    <span className='text-xs text-gray-400 dark:text-neutral-500 mt-1 block mb-4'>
      {props.description}
    </span>
  );
};
