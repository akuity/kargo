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
  borderRadius: 8
};

export const theme: ThemeConfig = {
  // ...token,
  token
};
