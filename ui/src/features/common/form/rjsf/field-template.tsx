import { Templates } from '@rjsf/antd';

import './style-overrides.module.less';

// https://github.com/rjsf-team/react-jsonschema-form/issues/2106#issuecomment-734990380
// @ts-expect-error when dependency doesn't provide a good way to define types
export const FieldTemplate = (props) => {
  return (
    // @ts-expect-error it is component
    <Templates.FieldTemplate {...props} formContext={{ wrapperStyle: { marginBottom: '0px' } }} />
  );
};
