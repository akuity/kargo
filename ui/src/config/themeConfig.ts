import { ThemeConfig } from 'antd/es/config-provider';
import { ComponentToken } from 'antd/es/menu/style';
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
  fontFamily: 'Poppins, sans-serif'
};

const Menu: Partial<ComponentToken> = {
  itemSelectedBg: 'lightgray'
};

export const themeConfig: ThemeConfig = {
  // ...token,
  token,
  components: {
    Menu
  }
};
