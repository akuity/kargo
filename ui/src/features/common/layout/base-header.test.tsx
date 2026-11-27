import { Breadcrumb } from 'antd';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, test } from 'vitest';

import { BaseHeader } from './base-header';

const items = [{ title: 'Projects' }, { title: 'demo' }, { title: 'Events' }];

const chevrons = (markup: string) => markup.match(/data-icon="chevron-right"/g)?.length ?? 0;

describe('BaseHeader', () => {
  test('separates breadcrumb items with a chevron', () => {
    const markup = renderToStaticMarkup(
      <BaseHeader>
        <Breadcrumb items={items} />
      </BaseHeader>
    );

    expect(chevrons(markup)).toBe(items.length - 1);
  });

  test('leaves breadcrumbs outside the header alone', () => {
    const markup = renderToStaticMarkup(<Breadcrumb items={items} />);

    expect(chevrons(markup)).toBe(0);
  });
});
