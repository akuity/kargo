import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { renderToStaticMarkup } from 'react-dom/server';
import { afterAll, beforeAll, describe, expect, test, vi } from 'vitest';

import { AuthContext } from '@ui/features/auth/context/auth-context';
import { JWTInfo } from '@ui/features/auth/jwt-utils';
import type { PublicConfig } from '@ui/gen/api/v2/models';
import { getGetPublicConfigQueryKey } from '@ui/gen/api/v2/system/system';

import { AccountCard } from './account-card';

const config: PublicConfig = {
  adminAccountEnabled: true,
  oidcConfig: { issuerUrl: 'https://kargo.example.com/dex' }
};

// The card as text, the way it reads: tags and React's text separators gone.
const render = (jwt: JWTInfo, publicConfig?: PublicConfig) => {
  const queryClient = new QueryClient();
  if (publicConfig) {
    queryClient.setQueryData(getGetPublicConfigQueryKey(), { data: publicConfig, status: 200 });
  }
  return renderToStaticMarkup(
    <QueryClientProvider client={queryClient}>
      <AuthContext.Provider
        value={{ isLoggedIn: true, login: () => {}, logout: () => {}, JWTInfo: jwt }}
      >
        <AccountCard />
      </AuthContext.Provider>
    </QueryClientProvider>
  ).replace(/<[^>]*>/g, '');
};

describe('AccountCard', () => {
  // Tests run without a DOM; the card only reads the host off the window.
  beforeAll(() => {
    vi.stubGlobal('window', { location: { host: 'kargo.example.com' } });
  });
  afterAll(() => {
    vi.unstubAllGlobals();
  });

  test('shows the built-in admin as such', () => {
    const text = render({ sub: 'admin', iss: 'https://kargo.example.com' }, config);
    expect(text).toContain('Admin account');
    expect(text).toContain('Signed in to kargo.example.com via built-in admin login');
  });

  test('does not take a single sign-on user called admin for the built-in admin', () => {
    const text = render({ sub: 'admin', iss: 'https://kargo.example.com/dex' }, config);
    expect(text).not.toContain('Admin account');
    expect(text).toContain('via single sign-on (kargo.example.com)');
  });

  test('shows the email beside the name', () => {
    const text = render(
      { sub: 'jane', iss: 'https://dex.example.com', name: 'Jane Doe', email: 'jane@example.com' },
      config
    );
    expect(text).toContain('Jane Doe');
    expect(text).toContain('jane@example.com');
  });

  test('does not repeat an email that is already the name', () => {
    const text = render({ sub: 'jane', iss: 'https://dex.example.com', email: 'jane@example.com' });
    expect(text.split('jane@example.com')).toHaveLength(2);
  });

  test('offers to log out', () => {
    expect(render({ sub: 'jane', iss: 'https://dex.example.com' })).toContain('Log out');
  });
});
