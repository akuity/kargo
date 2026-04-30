import { Descriptions } from 'antd';
import { DescriptionsItemType } from 'antd/es/descriptions';

export const ObjectDescription = (props: { data: object }) => {
  const items: DescriptionsItemType[] = [];

  for (const [key, value] of Object.entries(props.data)) {
    if (Array.isArray(value)) {
      if (value?.length > 0) {
        items.push({ key, label: key, children: value?.join(', ') });
      }
      continue;
    }

    if (value === null || value === undefined || typeof value === 'undefined') {
      continue;
    }

    if (typeof value === 'object') {
      items.push({ key, label: key, children: <ObjectDescription data={value} /> });
      continue;
    }

    items.push({
      key,
      label: key,
      children: `${value}`
    });
  }

  return <Descriptions bordered column={1} items={items} />;
};
