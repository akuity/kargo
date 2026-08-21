import { theme, ThemeConfig } from 'antd';
import { MapToken } from 'antd/es/theme/interface';

// Tokens shared by light and dark; mode-specific values live in the blocks below.
const sharedToken: Partial<MapToken> = {
  colorPrimary: '#30476c',
  fontSizeHeading1: 28,
  fontSizeHeading2: 24,
  fontSizeHeading3: 20,
  fontSizeHeading4: 18,
  fontSizeHeading5: 14,
  borderRadius: 8,
  fontFamily: 'Poppins, sans-serif'
};

// Light overrides. Also exported for vite.config.mts to seed build-time LESS vars.
export const token: Partial<MapToken> = {
  ...sharedToken,
  colorText: '#454545',
  colorBgLayout: '#f7f8fa'
};

// Dark overrides. The navy primary is too dark for the accents AntD derives from
// it on dark surfaces, so lighten it; links are text and need to be brighter
// still. colorBgBase lifts off pure black (page #141414, containers #282828).
const darkToken: Partial<MapToken> = {
  ...sharedToken,
  colorPrimary: '#506d9c',
  colorBgBase: '#141414',
  colorLink: '#7aa0d6',
  colorLinkHover: '#97b6e2',
  colorLinkActive: '#6690cc'
};

const sharedComponents: ThemeConfig['components'] = {
  Card: { borderRadius: 8 },
  Button: { contentFontSizeSM: 13 },
  Layout: { headerHeight: 50, headerPadding: '0 16px' },
  Menu: { itemHeight: 36 }
};

const lightComponents: ThemeConfig['components'] = {
  ...sharedComponents,
  Menu: {
    ...sharedComponents.Menu,
    itemActiveBg: '#ebedf1',
    itemSelectedBg: '#ebedf1',
    itemHoverBg: '#f8f9fb'
  },
  Layout: {
    ...sharedComponents.Layout,
    headerBg: '#fff'
  }
};

// Neutral-control hover/active accents: white text (blue reads too dark), with a
// softer translucent-white border (solid #fff looks harsh).
const darkAccentWhite = '#fff';
const darkAccentBorder = 'rgba(255, 255, 255, 0.45)';

const darkComponents: ThemeConfig['components'] = {
  ...sharedComponents,
  Button: {
    ...sharedComponents.Button,
    defaultHoverColor: darkAccentWhite,
    defaultHoverBorderColor: darkAccentBorder,
    defaultActiveColor: darkAccentWhite,
    defaultActiveBorderColor: darkAccentBorder
  },
  Tabs: {
    itemSelectedColor: darkAccentWhite,
    itemHoverColor: darkAccentWhite,
    itemActiveColor: darkAccentWhite,
    inkBarColor: darkAccentWhite
  },
  Menu: {
    ...sharedComponents.Menu,
    itemActiveBg: 'rgba(255, 255, 255, 0.08)',
    itemSelectedBg: 'rgba(255, 255, 255, 0.08)',
    itemHoverBg: 'rgba(255, 255, 255, 0.04)',
    itemSelectedColor: darkAccentWhite // else colorPrimary reads too dark
  },
  Layout: {
    ...sharedComponents.Layout,
    headerBg: '#1f1f1f'
  }
};

export const themeConfig = (isDark: boolean): ThemeConfig => ({
  algorithm: isDark ? theme.darkAlgorithm : theme.defaultAlgorithm,
  token: isDark ? darkToken : token,
  components: isDark ? darkComponents : lightComponents
});
