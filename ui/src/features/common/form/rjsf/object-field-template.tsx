import { Templates } from '@rjsf/antd';
import { Collapse } from 'antd';

// https://github.com/rjsf-team/react-jsonschema-form/issues/2106#issuecomment-734990380
// @ts-expect-error when dependency doesn't provide a good way to define types
export const ObjectFieldTemplate = (props) => {
  if (!Templates.ObjectFieldTemplate) {
    throw new Error('[BUG]: Templates.ObjectFieldTemplate is undefined');
  }

  if (props.title) {
    return (
      <Collapse
        items={[
          {
            children: <Templates.ObjectFieldTemplate {...props} />,
            label: props.title
          }
        ]}
      />
    );
  }

  return <Templates.ObjectFieldTemplate {...props} />;
};
