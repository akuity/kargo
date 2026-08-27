import { faDesktop, faMoon, faSun } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Card, Flex, Segmented, Switch, Typography } from 'antd';

import { useExtensionsContext } from '@ui/extensions/extensions-context';
import { useEmbeddedArgoCD } from '@ui/features/common/preferences/use-embedded-argocd';
import { ThemeMode } from '@ui/features/common/theme/theme-context';
import { useTheme } from '@ui/features/common/theme/use-theme';

// UI preferences; each is its own Card so more can stack below.
export const UISettings = () => {
  const { mode, setMode } = useTheme();
  const { argoCDExtension } = useExtensionsContext();
  const [embeddedArgoCD, setEmbeddedArgoCD] = useEmbeddedArgoCD();

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
      {Boolean(argoCDExtension) && (
        <Card type='inner' title='ArgoCD'>
          <Flex align='center' justify='space-between' gap={24} className='max-w-lg'>
            <div>
              <div className='font-medium'>Embedded ArgoCD</div>
              <Typography.Text type='secondary' className='text-xs'>
                Open ArgoCD applications inside Kargo. When off, links open the ArgoCD UI in a new
                tab.
              </Typography.Text>
            </div>
            <Switch checked={embeddedArgoCD} onChange={setEmbeddedArgoCD} />
          </Flex>
        </Card>
      )}
    </Flex>
  );
};
