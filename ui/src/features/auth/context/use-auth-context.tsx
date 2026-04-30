import React from 'react';

import { AuthContext } from './auth-context';

export const useAuthContext = () => {
  const ctx = React.useContext(AuthContext);

  if (ctx === null) {
    throw new Error(`useAuthContext must be used within a Provider`);
  }

  return ctx;
};
