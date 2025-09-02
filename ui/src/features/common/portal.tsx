import ErrorBoundary from 'antd/es/alert/ErrorBoundary';
import React from 'react';
import ReactDOM from 'react-dom';

type PortalProps = {
  children: React.ReactElement;
  container?: HTMLElement | null;
  disablePortal?: boolean;
};

export const Portal = ({
  container = document.body,
  children,
  disablePortal = false
}: PortalProps) => {
  if (disablePortal) {
    return children;
  }

  return (
    <ErrorBoundary>{container ? ReactDOM.createPortal(children, container) : null}</ErrorBoundary>
  );
};
