import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { ProtectedRoute } from './protected-route';

const state = vi.hoisted(() => ({ isLoading: true, isLoggedIn: false, skipAuth: false }));
vi.mock('@ui/gen/api/v2/system/system', () => ({
  useGetPublicConfig: () => ({
    isLoading: state.isLoading,
    data: { data: { skipAuth: state.skipAuth } }
  })
}));
vi.mock('./context/use-auth-context', () => ({ useAuthContext: () => state }));
vi.mock('react-router-dom', () => ({
  Navigate: ({ to }: { to: string }) => to,
  Outlet: () => null
}));
vi.mock('@ui/features/promotion-directives/registry/context/registry-context-provider', () => ({
  PromotionDirectivesRegistryContextProvider: () => 'protected children mounted'
}));
vi.mock('@ui/features/common/modal/modal-context-provider', () => ({
  ModalContextProvider: () => null
}));

describe('protected route bootstrap', () => {
  beforeEach(() => {
    Object.assign(state, { isLoading: true, isLoggedIn: false, skipAuth: false });
    vi.stubGlobal('window', { location: { pathname: '/project/example/stage/dev' } });
  });

  it('does not mount protected request providers while public config is loading', () => {
    expect(renderToStaticMarkup(createElement(ProtectedRoute))).toBe('');
  });

  it('redirects the original deep link once authentication is known to be required', () => {
    state.isLoading = false;
    expect(renderToStaticMarkup(createElement(ProtectedRoute))).toContain(
      '/login?redirectTo=/project/example/stage/dev'
    );
  });

  it.each(['isLoggedIn', 'skipAuth'] as const)('allows protected children for %s', (field) => {
    state.isLoading = false;
    state[field] = true;
    expect(renderToStaticMarkup(createElement(ProtectedRoute))).toBe('protected children mounted');
  });
});
