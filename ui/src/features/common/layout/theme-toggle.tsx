import { faCircleHalfStroke, faDesktop, faMoon, faSun } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Dropdown, MenuProps, Tooltip } from 'antd';

import { ThemePreference } from '@ui/features/theme/types';
import { useThemeContext } from '@ui/features/theme/use-theme-context';

import * as styles from './main-layout.module.less';

const labels: Record<ThemePreference, string> = {
  system: 'System',
  light: 'Light',
  dark: 'Dark'
};

export const ThemeToggle = () => {
  const { preference, setPreference } = useThemeContext();

  const items: MenuProps['items'] = [
    {
      key: 'system',
      label: 'System',
      icon: <FontAwesomeIcon icon={faDesktop} />
    },
    {
      key: 'light',
      label: 'Light',
      icon: <FontAwesomeIcon icon={faSun} />
    },
    {
      key: 'dark',
      label: 'Dark',
      icon: <FontAwesomeIcon icon={faMoon} />
    }
  ];

  return (
    <Dropdown
      menu={{
        items,
        selectable: true,
        selectedKeys: [preference],
        onClick: ({ key }) => setPreference(key as ThemePreference)
      }}
      placement='top'
      trigger={['click']}
    >
      <Tooltip title={`Theme: ${labels[preference]}`} placement='right'>
        <Button
          className={styles.themeButton}
          type='text'
          icon={<FontAwesomeIcon icon={faCircleHalfStroke} />}
        />
      </Tooltip>
    </Dropdown>
  );
};
