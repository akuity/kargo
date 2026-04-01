import React from 'react';

export type DocumentTitleContextType = {
  appName?: string;
};

export const DocumentTitleContext = React.createContext<DocumentTitleContextType>({
  appName: 'Kargo'
});
