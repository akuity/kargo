import { Templates } from '@rjsf/antd';
import { FieldTemplateProps } from '@rjsf/utils';

export const FieldTemplate = (props: FieldTemplateProps) => {
  return (
    // @ts-expect-error it is component
    <Templates.FieldTemplate {...props} formContext={{ wrapperStyle: { marginBottom: '0px' } }} />
  );
};
