import classNames from 'classnames';
import { PropsWithChildren } from 'react';

import { PluginErrorBoundary } from './error-boundary';

export const DeepLinkPromotion = ({
  children,
  className
}: PropsWithChildren<{ className?: string }>) => (
  <PluginErrorBoundary>
    <div className={classNames(className, 'bg-gray-100 px-2 py-1 rounded-md text-sm')}>
      {children}
    </div>
  </PluginErrorBoundary>
);
