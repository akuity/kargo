import { faExternalLink } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Dropdown, ButtonProps, Button } from 'antd';
import React from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';
import z from 'zod';

import { withBasePath } from '@ui/config/base-path';
import { ARGOCD_CONTEXT_KEY, SHARD_LABEL_KEY } from '@ui/config/labels';
import { paths } from '@ui/config/paths';
import { useExtensionsContext } from '@ui/extensions/extensions-context';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { Health, Stage } from '@ui/gen/api/v2/models';

import { useDictionaryContext } from '../context/dictionary-context';

type ArgoCDLinkProps = React.PropsWithChildren<{
  stage: Stage;
  externalLinksOnly?: boolean;
  buttonProps: ButtonProps;
}>;

export const ArgoCDLink = ({
  buttonProps,
  children,
  externalLinksOnly,
  stage
}: ArgoCDLinkProps) => {
  const navigate = useNavigate();
  const { name: projectName } = useParams();
  const { argoCDExtension } = useExtensionsContext();
  const dictionaryContext = useDictionaryContext();

  const shardKey = stage?.metadata?.labels?.[SHARD_LABEL_KEY] || '';
  // Remove trailing slash if present
  const argoCDShardURL = dictionaryContext?.argocdShards?.[shardKey]?.url?.replace(/\/$/, '');
  const isExtensionArgoCD = Boolean(argoCDExtension) && !externalLinksOnly;

  const argoCDApps = React.useMemo(() => {
    const rawValues = stage.metadata?.annotations?.[ARGOCD_CONTEXT_KEY];

    try {
      return rawValues ? argoCDContextSchema.parse(JSON.parse(rawValues)) : [];
    } catch (e) {
      // eslint-disable-next-line no-console
      console.error(e);
      return [];
    }
  }, [stage]);

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
  const argoCDHref = React.useCallback(
    (link: ArgoCDContext) =>
      isExtensionArgoCD
        ? withBasePath(extensionPath(link))
        : `${argoCDShardURL}/applications/${link.namespace}/${link.name}`,
    [isExtensionArgoCD, extensionPath, argoCDShardURL]
  );

  // A plain click on the in-app extension route stays SPA navigation; modified
  // clicks fall through to the browser's native new-tab handling.
  const openArgoCD = React.useCallback(
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

  if (argoCDApps.length === 0 || !argoCDShardURL) {
    return null;
  }

  if (argoCDApps.length === 1) {
    return (
      <Button
        href={argoCDHref(argoCDApps[0])}
        target={isExtensionArgoCD ? undefined : '_blank'}
        rel={isExtensionArgoCD ? undefined : 'noopener noreferrer'}
        onClick={(e) => openArgoCD(e, argoCDApps[0])}
        {...buttonProps}
      >
        {children}
      </Button>
    );
  }

  return (
    <Dropdown
      trigger={['click']}
      menu={{
        style: { maxHeight: '278px', overflowY: 'auto' },
        items: argoCDApps.map((app, idx) => {
          const status = stage.status?.health?.output
            ? getStatusFromHealthOutput(stage.status.health.output, app.name)
            : undefined;

          return {
            key: idx,
            label: (
              <a
                href={argoCDHref(app)}
                target={isExtensionArgoCD ? undefined : '_blank'}
                rel={isExtensionArgoCD ? undefined : 'noopener noreferrer'}
                onClick={(e) => openArgoCD(e, app)}
              >
                {status && <HealthStatusIcon health={status} className='mr-2' />}
                {`${app.name} - ${app.namespace}`}
                {!isExtensionArgoCD && (
                  <FontAwesomeIcon icon={faExternalLink} className='text-xs ml-2' />
                )}
              </a>
            )
          };
        })
      }}
    >
      <Button {...buttonProps}>{children}</Button>
    </Dropdown>
  );
};

const argoCDContextSchema = z.array(
  z.object({
    name: z.string(),
    namespace: z.string()
  })
);

type ArgoCDContext = z.infer<typeof argoCDContextSchema>[number];

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
