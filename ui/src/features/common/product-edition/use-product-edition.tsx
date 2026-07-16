import React from 'react';

import { ProductEditionContext } from './product-edition-context';

export const useProductEdition = () => {
  const context = React.useContext(ProductEditionContext);

  if (context === null) {
    throw new Error('useProductEdition must be used within a ProductEditionContextProvider');
  }

  return context;
};
