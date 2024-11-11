import { Templates } from '@rjsf/antd';
import { ObjectFieldTemplateProps } from '@rjsf/utils';
import { Collapse } from 'antd';
import { useMemo } from 'react';

export const ObjectFieldTemplate = (props: ObjectFieldTemplateProps) => {
  if (!Templates.ObjectFieldTemplate) {
    throw new Error('[BUG]: Templates.ObjectFieldTemplate is undefined');
  }

  const orderedProperties = useMemo(
    () =>
      props.properties.sort((a, b) => {
        const aRequired = props.schema.required?.includes(a.name);
        const bRequired = props.schema.required?.includes(b.name);

        if (aRequired) {
          return -1;
        }

        if (bRequired) {
          return 1;
        }

        return 0;
      }),
    [props.properties, props.schema.required]
  );

  if (props.title) {
    return (
      <Collapse
        items={[
          {
            children: <Templates.ObjectFieldTemplate {...props} properties={orderedProperties} />,
            label: props.title
          }
        ]}
      />
    );
  }

  return <Templates.ObjectFieldTemplate {...props} properties={orderedProperties} />;
};
