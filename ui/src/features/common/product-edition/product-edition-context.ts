import React from 'react';

import { GithubComAkuityKargoPkgXEditionEdition as GeneratedProductEdition } from '@ui/gen/api/v2/models/githubComAkuityKargoPkgXEditionEdition';

export const ProductEdition = GeneratedProductEdition;
export type ProductEdition = GeneratedProductEdition;

export interface ProductEditionContextType {
  edition: ProductEdition;
  isEnterprise: boolean;
  isLoading: boolean;
}

export const ProductEditionContext = React.createContext<ProductEditionContextType | null>(null);
