import { ThemeConfig } from 'antd/es/config-provider';

export const theme: ThemeConfig = {
  components: {
    Button: {
      colorPrimary: '#44505f',
      colorPrimaryBgHover: '#626f7e',
      colorPrimaryHover: '#626f7e',
      colorPrimaryActive: '#626f7e',
      boxShadow: '0',
      borderRadius: 100,
      borderRadiusSM: 100,
      borderRadiusLG: 100
    },
    Switch: {
      colorPrimary: '#44505f',
      colorPrimaryHover: '#626f7e',
      colorPrimaryActive: '#626f7e'
    }
  }
};
