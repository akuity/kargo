import { theme } from 'antd';
import { ThemeConfig } from 'antd/es/config-provider';
import { MapToken } from 'antd/es/theme/interface';

import type { ResolvedTheme } from '@ui/features/theme/types';

export const token: Partial<MapToken> = {
  colorPrimary: '#30476c',
  fontSizeHeading1: 28,
  fontSizeHeading2: 24,
  fontSizeHeading3: 20,
  fontSizeHeading4: 18,
  fontSizeHeading5: 14,
  borderRadius: 8,
  fontFamily: 'Poppins, sans-serif'
};

const lightToken: Partial<MapToken> = {
  colorText: '#454545',
  colorBgLayout: '#f7f8fa',
  colorBgContainer: '#ffffff',
  colorBorderSecondary: '#eef2f6'
};

const darkToken: Partial<MapToken> = {
  colorText: '#d7dee7',
  colorTextSecondary: '#9cadbd',
  colorBgLayout: '#0f141a',
  colorBgContainer: '#161d24',
  colorBorder: '#2a333d',
  colorBorderSecondary: '#222b34'
};

export const getThemeConfig = (mode: ResolvedTheme): ThemeConfig => {
  const isDark = mode === 'dark';

  return {
    cssVar: true,
    algorithm: isDark ? theme.darkAlgorithm : theme.defaultAlgorithm,
    token: {
      ...token,
      ...(isDark ? darkToken : lightToken)
    },
    components: {
      Menu: {
        itemActiveBg: isDark ? '#1d2630' : '#ebedf1',
        itemSelectedBg: isDark ? '#1d2630' : '#ebedf1',
        itemHoverBg: isDark ? '#18212a' : '#f8f9fb',
        itemHeight: 36
      },
      Layout: {
        headerBg: isDark ? '#161d24' : '#fff',
        headerHeight: 50,
        headerPadding: '0 16px'
      },
      Card: {
        borderRadius: 8
      },
      Button: {
        contentFontSizeSM: 13
      }
    }
  };
};
