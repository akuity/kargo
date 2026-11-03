import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ModalProps } from 'antd';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';

import { CreateProjectModal } from './create-project-modal';

const mutation = vi.hoisted(() => ({ isPending: false, mutate: vi.fn() }));
vi.mock('@ui/gen/api/v2/resources/resources', () => ({ useCreateResource: () => mutation }));
vi.mock('@ui/features/common/code-editor/yaml-editor', () => ({ YamlEditor: () => null }));
vi.mock('antd', async (importOriginal) => {
  const actual = await importOriginal<typeof import('antd')>();
  return {
    ...actual,
    // Render the actual AntD submit button without the browser-only modal portal.
    Modal: ({ okButtonProps, okText }: ModalProps) =>
      createElement(actual.Button, okButtonProps, okText)
  };
});

describe('Quick create submission accessibility', () => {
  it.each([true, false])('keeps the Create name with pending=%s', (isPending) => {
    mutation.isPending = isPending;
    const client = new QueryClient();
    const html = renderToStaticMarkup(
      createElement(
        QueryClientProvider,
        { client },
        createElement(CreateProjectModal, { visible: true, hide: vi.fn() })
      )
    );
    // The loading icon has its own accessible label. An explicit button name
    // must take precedence even while that icon is entering or leaving.
    expect(html).toMatch(/<button[^>]*aria-label="Create"/);
    expect(html).toContain(`aria-busy="${isPending}"`);
    if (isPending) {
      expect(html).toMatch(/<button[^>]*disabled=""/);
      expect(html).toContain('aria-label="loading"');
    } else {
      expect(html).not.toMatch(/<button[^>]*disabled=/);
    }
    client.clear();
  });
});
