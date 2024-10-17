import { Templates } from '@rjsf/antd';
import { ObjectFieldTemplateProps } from '@rjsf/utils';
import { Collapse } from 'antd';

export const ObjectFieldTemplate = (props: ObjectFieldTemplateProps) => {
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
