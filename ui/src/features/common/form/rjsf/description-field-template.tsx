// @ts-expect-error when dependency doesn't provide a good way to define types
export const DescriptionFieldTemplate = (props) => (
  <span className='text-xs text-gray-400 mt-1 block mb-4'>{props.description}</span>
);
