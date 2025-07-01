import React from 'react';

export type ModalItem = {
  component: ModalComponent;
  visible: boolean;
};

export interface ModalContextValue {
  show: (key: string, element: ModalComponent) => void;
  hide: (key: string) => void;
}

export const ModalContext = React.createContext<ModalContextValue | null>(null);

export interface ModalComponentProps {
  hide: () => void;
  visible: boolean;
}

export type ModalComponent = React.FC<ModalComponentProps>;
