import ErrorBoundary from 'antd/es/alert/ErrorBoundary';
import React from 'react';
import ReactDOM from 'react-dom';

type PortalProps = {
  children: React.ReactElement;
  container?: HTMLElement | null;
  disablePortal?: boolean;
};

export const Portal = ({
  container = typeof document !== 'undefined' ? document.body : null,
  children,
  disablePortal = false
}: PortalProps) => {
  if (disablePortal) {
    return children;
  }

  if (!container) {
    return null;
  }

  return <ErrorBoundary>{ReactDOM.createPortal(children, container)}</ErrorBoundary>;
};
