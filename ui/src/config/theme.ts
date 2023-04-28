import { ThemeConfig } from 'antd/es/config-provider';
import { MapToken } from 'antd/es/theme/interface';

export const token: Partial<MapToken> = {
  colorPrimary: 'rgb(29, 50, 82)',
  fontSizeHeading1: 28,
  fontSizeHeading2: 24,
  colorText: '#454545',
  borderRadius: 8
};

export const theme: ThemeConfig = {
  // ...token,
  token,
  }
};
