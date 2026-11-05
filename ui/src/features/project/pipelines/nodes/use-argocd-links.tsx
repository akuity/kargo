import { ItemType } from 'antd/es/menu/interface';
import React from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { withBasePath } from '@ui/config/base-path';
import { ARGOCD_CONTEXT_KEY, SHARD_LABEL_KEY } from '@ui/config/labels';
import { paths } from '@ui/config/paths';
import { useExtensionsContext } from '@ui/extensions/extensions-context';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { useEmbeddedArgoCD } from '@ui/features/common/preferences/use-embedded-argocd';
import { Health, Stage } from '@ui/gen/api/v2/models';

import { useDictionaryContext } from '../context/dictionary-context';

import { ArgoCDContext, argoCDAppKey, parseArgoCDContext } from './argocd-link-utils';

import './stage-node.less';

// useArgoCDLinks answers where the Argo CD Applications behind a Stage live
// and how to open them, so a button and a menu entry elsewhere can both be
// built from the same answer. It reports no Applications at all when the
// shard they belong to has no URL -- there is nothing to link to.
export const useArgoCDLinks = (stage: Stage, externalLinksOnly?: boolean) => {
  const navigate = useNavigate();
  const { name: projectName } = useParams();
  const { argoCDExtension } = useExtensionsContext();
  const dictionaryContext = useDictionaryContext();
  const [embeddedArgoCD] = useEmbeddedArgoCD();

  const shardKey = stage?.metadata?.labels?.[SHARD_LABEL_KEY] || '';
  // Remove trailing slash if present
  const argoCDShardURL = dictionaryContext?.argocdShards?.[shardKey]?.url?.replace(/\/$/, '');
  const isExtensionArgoCD = Boolean(argoCDExtension) && !externalLinksOnly && embeddedArgoCD;

  const apps = React.useMemo(() => {
    if (!argoCDShardURL) {
      return [];
    }

    try {
      return parseArgoCDContext(stage.metadata?.annotations?.[ARGOCD_CONTEXT_KEY]);
    } catch (e) {
      // eslint-disable-next-line no-console
      console.error(e);
      return [];
    }
  }, [stage, argoCDShardURL]);

  // Router-relative path for the in-app extension route (no basePath -- navigate
  // prepends the router basename itself).
  const extensionPath = React.useCallback(
    (link: ArgoCDContext) =>
      generatePath(paths.projectArgoCDExtension, {
        name: projectName,
        namespace: link.namespace,
        appName: link.name,
        stageName: stage.metadata?.name
      }),
    [projectName, stage]
  );

  // Browser href carrying the real destination, so cmd/ctrl-click and "open in
  // new tab" work natively. withBasePath re-adds the prefix for the SPA route.
  const href = React.useCallback(
    (link: ArgoCDContext) =>
      isExtensionArgoCD
        ? withBasePath(extensionPath(link))
        : `${argoCDShardURL}/applications/${link.namespace}/${link.name}`,
    [isExtensionArgoCD, extensionPath, argoCDShardURL]
  );

  // A plain click on the in-app extension route stays SPA navigation; modified
  // clicks fall through to the browser's native new-tab handling.
  const open = React.useCallback(
    (e: React.MouseEvent, link: ArgoCDContext) => {
      if (!isExtensionArgoCD) {
        return;
      }
      if (e.metaKey || e.ctrlKey || e.shiftKey || e.button === 1) {
        return;
      }
      e.preventDefault();
      navigate(extensionPath(link));
    },
    [isExtensionArgoCD, navigate, extensionPath]
  );

  const healthOf = React.useCallback(
    (app: string) =>
      stage.status?.health?.output
        ? getStatusFromHealthOutput(stage.status.health.output, app)
        : undefined,
    [stage]
  );

  return { apps, isExtensionArgoCD, href, open, healthOf };
};

// useArgoCDMenuItems describes the same Applications as entries for a menu
// that carries other actions too. A Stage with several of them gets a
// submenu, so the menu it joins stays one line longer, not several.
export const useArgoCDMenuItems = (stage: Stage): ItemType[] => {
  const { apps, isExtensionArgoCD, href, open, healthOf } = useArgoCDLinks(stage);

  if (apps.length === 0) {
    return [];
  }

  const link = (app: ArgoCDContext, label: React.ReactNode) => (
    <a
      className='stage-node__menu-link'
      href={href(app)}
      target={isExtensionArgoCD ? undefined : '_blank'}
      rel={isExtensionArgoCD ? undefined : 'noopener noreferrer'}
      onClick={(e) => open(e, app)}
    >
      {label}
    </a>
  );

  const icon = (
    <span className='stage-node__menu-icon'>
      <img src={withBasePath('/argo-logo.svg')} alt='' style={{ width: '12px' }} />
    </span>
  );

  if (apps.length === 1) {
    return [{ key: 'argocd', icon, label: link(apps[0], 'Argo CD') }];
  }

  return [
    {
      key: 'argocd',
      icon,
      label: 'Argo CD',
      children: apps.map((app) => ({
        key: argoCDAppKey(app),
        label: link(app, argoCDAppLabel(app, healthOf(app.name)))
      }))
    }
  ];
};

// argoCDAppLabel names one Application, with its health when the Stage
// reported any.
export const argoCDAppLabel = (app: ArgoCDContext, health?: Health) => (
  <>
    {health && <HealthStatusIcon health={health} className='mr-2' />}
    {`${app.name} - ${app.namespace}`}
  </>
);

const getStatusFromHealthOutput = (healthOutput: unknown, app: string): Health | undefined => {
  try {
    type HealthEntry = { applicationStatuses?: Array<{ Name: string; health?: Health }> };
    const outputs = healthOutput as HealthEntry[];
    const appStatus = outputs
      .flatMap((item) => item.applicationStatuses ?? [])
      .find((status) => status.Name === app);
    return appStatus?.health;
  } catch {
    return undefined;
  }
};
