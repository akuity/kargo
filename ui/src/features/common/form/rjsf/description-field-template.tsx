import { DescriptionFieldProps } from '@rjsf/utils';

export const DescriptionFieldTemplate = (props: DescriptionFieldProps) => (
  <span className='text-xs text-gray-400 mt-1 block mb-4'>{props.description}</span>
);
