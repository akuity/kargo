import { describe, expect, it } from 'vitest';

import { getGitCommitURL } from './open-container-initiative-utils';

describe('getGitCommitURL', () => {
  const revision = 'abc1234';

  const cases: { name: string; url: string; expected: string }[] = [
    // GitHub — hosted
    {
      name: 'github HTTPS',
      url: 'https://github.com/akuity/kargo.git',
      expected: 'https://github.com/akuity/kargo/commit/abc1234'
    },
    {
      name: 'github SSH',
      url: 'git@github.com:akuity/kargo.git',
      expected: 'https://github.com/akuity/kargo/commit/abc1234'
    },
    // GitHub Enterprise — self-hosted
    {
      name: 'github enterprise custom domain',
      url: 'https://github.internal.net/akuity/kargo.git',
      expected: 'https://github.internal.net/akuity/kargo/commit/abc1234'
    },
    // GitLab — hosted
    {
      name: 'gitlab HTTPS',
      url: 'https://gitlab.com/akuity/kargo.git',
      expected: 'https://gitlab.com/akuity/kargo/-/commit/abc1234'
    },
    {
      name: 'gitlab SSH',
      url: 'git@gitlab.com:akuity/kargo.git',
      expected: 'https://gitlab.com/akuity/kargo/-/commit/abc1234'
    },
    // GitLab — self-hosted
    {
      name: 'gitlab self-managed custom domain',
      url: 'https://gitlab.internal.net/akuity/kargo.git',
      expected: 'https://gitlab.internal.net/akuity/kargo/-/commit/abc1234'
    },
    {
      name: 'gitlab self-managed SSH',
      url: 'git@gitlab.internal.net:akuity/kargo.git',
      expected: 'https://gitlab.internal.net/akuity/kargo/-/commit/abc1234'
    },
    // Bitbucket — hosted
    {
      name: 'bitbucket HTTPS',
      url: 'https://bitbucket.org/akuity/kargo.git',
      expected: 'https://bitbucket.org/akuity/kargo/commits/abc1234'
    },
    {
      name: 'bitbucket SSH',
      url: 'git@bitbucket.org:akuity/kargo.git',
      expected: 'https://bitbucket.org/akuity/kargo/commits/abc1234'
    },
    // Bitbucket Data Center — self-hosted
    {
      name: 'bitbucket data center custom domain',
      url: 'https://bitbucket.internal.net/akuity/kargo.git',
      expected: 'https://bitbucket.internal.net/akuity/kargo/commits/abc1234'
    },
    {
      name: 'bitbucket data center SSH',
      url: 'git@bitbucket.internal.net:akuity/kargo.git',
      expected: 'https://bitbucket.internal.net/akuity/kargo/commits/abc1234'
    },
    // Unknown provider — fall back to original url
    {
      name: 'unknown provider returns original url',
      url: 'https://git.example.com/akuity/kargo.git',
      expected: 'https://git.example.com/akuity/kargo.git'
    }
  ];

  for (const tc of cases) {
    it(tc.name, () => {
      expect(getGitCommitURL(tc.url, revision)).toBe(tc.expected);
    });
  }
});
