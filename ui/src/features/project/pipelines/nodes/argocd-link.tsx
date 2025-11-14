import { faExternalLink } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Dropdown, ButtonProps, Button } from 'antd';
import React from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';
import z from 'zod';

import { paths } from '@ui/config/paths';
import { useExtensionsContext } from '@ui/extensions/extensions-context';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { useDictionaryContext } from '../context/dictionary-context';

const ARGOCD_CONTEXT_KEY = 'kargo.akuity.io/argocd-context';
const SHARD_LABEL_KEY = 'kargo.akuity.io/shard';

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

  const shardKey = stage?.metadata?.labels[SHARD_LABEL_KEY] || '';
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

  const openArgoCD = React.useCallback(
    (link: ArgoCDContext) => {
      if (isExtensionArgoCD) {
        navigate(
          generatePath(paths.projectArgoCDExtension, {
            name: projectName,
            namespace: link.namespace,
            appName: link.name
          })
        );
      } else {
        window.open(
          `${argoCDShardURL}/applications/${link.namespace}/${link.name}`,
          '_blank',
          'noopener noreferrer'
        );
      }
    },
    [isExtensionArgoCD, navigate, projectName, argoCDShardURL]
  );

  if (argoCDApps.length === 0 || !argoCDShardURL) {
    return null;
  }

  if (argoCDApps.length === 1) {
    return (
      <Button onClick={() => openArgoCD(argoCDApps[0])} {...buttonProps}>
        {children}
      </Button>
    );
  }

  return (
    <Dropdown
      trigger={['click']}
      menu={{
        items: argoCDApps.map((app, idx) => {
          return {
            key: idx,
            label: (
              <a
                href='#'
                onClick={(e) => {
                  e.preventDefault();
                  openArgoCD(app);
                }}
              >
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
