import { faChevronDown } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Dropdown, Space, Tooltip } from 'antd';

import { getPromotionHealthCheckConfig } from './plugin-helper';
import { PluginsInstallation } from './plugin-interfaces';

const argocdPlugin: PluginsInstallation = {
  DeepLinkPlugin: {
    Promotion: {
      shouldRender(opts) {
        return (
          Boolean(opts?.isLatestPromotion) &&
          Boolean(opts.promotion?.spec?.steps?.find((step) => step?.uses === 'argocd-update')) &&
          (opts.promotion?.status?.healthChecks?.length || 0) > 0
        );
      },
      render(props) {
        if (!props.unstable_argocdShardUrl) {
          return (
            <Tooltip title='Unknown ArgoCD shard' className='cursor-pointer text-xs text-gray-400'>
              ArgoCD
            </Tooltip>
          );
        }

        // argocd shards sometimes might have base path included
        // in those cases, we must not omit those pathname in order to have valid
        const unstable_argocdShardUrl = props.unstable_argocdShardUrl.endsWith('/')
          ? props.unstable_argocdShardUrl.slice(0, -1)
          : props.unstable_argocdShardUrl;

        const healthChecks = (props.promotion?.status?.healthChecks || []).filter(
          (hc) => hc?.uses === 'argocd-update'
        );

        // health checks contains nested apps
        // ie. healthChecks:
        // - uses: argocd-update
        //   config:
        //     apps:
        //       - name: app
        //         namespace: ns
        const apps = [];

        for (const healthCheck of healthChecks) {
          const healthCheckConfig = getPromotionHealthCheckConfig(healthCheck);

          // @ts-expect-error we don't have type but as long as we are sure whats coming in.. its safe to assume
          for (const app of healthCheckConfig?.apps || []) {
            apps.push(app);
          }
        }

        if (apps.length === 1) {
          return (
            <a
              target='_blank'
              href={`${unstable_argocdShardUrl}/applications/${apps[0].namespace}/${apps[0].name}`}
            >
              ArgoCD
            </a>
          );
        }

        return (
          <Dropdown
            menu={{
              items: apps.map((app, idx) => ({
                key: idx,
                label: (
                  <a
                    target='_blank'
                    href={`${unstable_argocdShardUrl}/applications/${app.namespace}/${app.name}`}
                  >
                    {app.namespace} - {app.name}
                  </a>
                )
              }))
            }}
          >
            <a onClick={(e) => e.preventDefault()}>
              <Space>
                ArgoCD
                <FontAwesomeIcon icon={faChevronDown} className='text-xs' />
              </Space>
            </a>
          </Dropdown>
        );
      }
    }
  }
};

export default argocdPlugin;
