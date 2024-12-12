import classNames from 'classnames';
import { PropsWithChildren } from 'react';

export const DeepLinkPromotion = ({
  children,
  className
}: PropsWithChildren<{ className?: string }>) => (
  <div className={classNames(className, 'bg-gray-100 px-2 py-1 rounded-md text-sm')}>
    {children}
  </div>
);
