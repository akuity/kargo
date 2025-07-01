import classNames from 'classnames';
import { PropsWithChildren } from 'react';

import { PluginErrorBoundary } from './error-boundary';

export const DeepLinkPromotionStep = ({
  children,
  className
}: PropsWithChildren<{ className?: string }>) => {
  return (
    <PluginErrorBoundary>
      <div
        className={classNames(
          className,
          'bg-gray-100 px-2 py-1 rounded-md text-sm w-fit flex gap-2'
        )}
        onClick={(e) => {
          // prevent opening the collapsible menu
          e.stopPropagation();
        }}
      >
        {children}
      </div>
    </PluginErrorBoundary>
  );
};
