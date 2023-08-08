import React from 'react';
import ReactDOM from 'react-dom';

type PortalProps = {
  children: React.ReactElement;
  container?: HTMLElement;
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

  return container ? ReactDOM.createPortal(children, container) : null;
};
