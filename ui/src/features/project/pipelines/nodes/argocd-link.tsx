import { faExternalLink } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Dropdown, ButtonProps, Button } from 'antd';
import React from 'react';

import { Stage } from '@ui/gen/api/v2/models';

import { argoCDAppKey } from './argocd-link-utils';
import { argoCDAppLabel, useArgoCDLinks } from './use-argocd-links';

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
  const { apps, isExtensionArgoCD, href, open, healthOf } = useArgoCDLinks(
    stage,
    externalLinksOnly
  );

  if (apps.length === 0) {
    return null;
  }

  if (apps.length === 1) {
    return (
      <Button
        href={href(apps[0])}
        target={isExtensionArgoCD ? undefined : '_blank'}
        rel={isExtensionArgoCD ? undefined : 'noopener noreferrer'}
        onClick={(e) => open(e, apps[0])}
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
        items: apps.map((app) => ({
          key: argoCDAppKey(app),
          label: (
            <a
              href={href(app)}
              target={isExtensionArgoCD ? undefined : '_blank'}
              rel={isExtensionArgoCD ? undefined : 'noopener noreferrer'}
              onClick={(e) => open(e, app)}
            >
              {argoCDAppLabel(app, healthOf(app.name))}
              {!isExtensionArgoCD && (
                <FontAwesomeIcon icon={faExternalLink} className='text-xs ml-2' />
              )}
            </a>
          )
        }))
      }}
    >
      <Button {...buttonProps}>{children}</Button>
    </Dropdown>
  );
};
