import { faApple, faLinux, faWindows } from '@fortawesome/free-brands-svg-icons';

import { Platform } from './types';

export const GITHUB_RELEASES_URL = 'https://github.com/akuity/kargo/releases';

const latestUrl = (filename: string) => `${GITHUB_RELEASES_URL}/latest/download/${filename}`;

export const PLATFORMS: Platform[] = [
  {
    title: 'Mac',
    icon: faApple,
    links: [
      {
        title: 'Apple Silicon',
        getUrl: (r) => r?.cliBinaries.darwin.arm64 ?? latestUrl('kargo-darwin-arm64')
      },
      {
        title: 'Intel',
        getUrl: (r) => r?.cliBinaries.darwin.amd64 ?? latestUrl('kargo-darwin-amd64')
      }
    ]
  },
  {
    title: 'Windows',
    icon: faWindows,
    links: [
      {
        title: 'ARM',
        getUrl: (r) => r?.cliBinaries.windows.arm64 ?? latestUrl('kargo-windows-arm64.exe')
      },
      {
        title: 'x86',
        getUrl: (r) => r?.cliBinaries.windows.amd64 ?? latestUrl('kargo-windows-amd64.exe')
      }
    ]
  },
  {
    title: 'Linux',
    icon: faLinux,
    links: [
      {
        title: 'ARM',
        getUrl: (r) => r?.cliBinaries.linux.arm64 ?? latestUrl('kargo-linux-arm64')
      },
      {
        title: 'x86',
        getUrl: (r) => r?.cliBinaries.linux.amd64 ?? latestUrl('kargo-linux-amd64')
      }
    ]
  }
];
