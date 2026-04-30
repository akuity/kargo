import { ThemeConfig } from 'antd/es/config-provider';
import { MapToken } from 'antd/es/theme/interface';

export const token: Partial<MapToken> = {
  colorPrimary: '#30476c',
  fontSizeHeading1: 28,
  fontSizeHeading2: 24,
  fontSizeHeading3: 20,
  fontSizeHeading4: 18,
  fontSizeHeading5: 14,
  colorText: '#454545',
  borderRadius: 8,
  fontFamily: 'Poppins, sans-serif',
  colorBgLayout: '#f7f8fa'
};

export const themeConfig: ThemeConfig = {
  // ...token,
  token,
  components: {
    Menu: {
      itemActiveBg: '#ebedf1',
      itemSelectedBg: '#ebedf1',
      itemHoverBg: '#f8f9fb',
      itemHeight: 36
    },
    Layout: {
      headerBg: '#fff',
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
