import classNames from 'classnames';

export const SmallLabel = ({
  children,
  className
}: {
  children: React.ReactNode;
  className?: string;
}) => <div className={classNames('text-xs text-gray-500 font-medium', className)}>{children}</div>;
