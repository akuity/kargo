import { faDesktop, faMoon, faSun } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Card, Flex, Segmented, Typography } from 'antd';

import { ThemeMode } from '@ui/features/common/theme/theme-context';
import { useTheme } from '@ui/features/common/theme/use-theme';

// UI preferences; each is its own Card so more can stack below.
export const UISettings = () => {
  const { mode, setMode } = useTheme();

  return (
    <Flex vertical gap={16}>
      <Card type='inner' title='Theme'>
        <Flex align='center' justify='space-between' gap={24} className='max-w-lg'>
          <div>
            <div className='font-medium'>Mode</div>
            <Typography.Text type='secondary' className='text-xs'>
              Choose light, dark, or follow your system setting.
            </Typography.Text>
          </div>
          <Segmented<ThemeMode>
            value={mode}
            onChange={setMode}
            options={[
              { label: 'Light', value: 'light', icon: <FontAwesomeIcon icon={faSun} /> },
              { label: 'System', value: 'system', icon: <FontAwesomeIcon icon={faDesktop} /> },
              { label: 'Dark', value: 'dark', icon: <FontAwesomeIcon icon={faMoon} /> }
            ]}
          />
        </Flex>
      </Card>
    </Flex>
  );
};
