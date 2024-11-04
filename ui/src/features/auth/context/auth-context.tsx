import React from 'react';

import { JWTInfo } from '../utils';

export interface AuthContextType {
  isLoggedIn: boolean;
  login: (token: string, authToken?: string) => void;
  logout: () => void;
  JWTInfo?: JWTInfo | null;
}

export const AuthContext = React.createContext<AuthContextType | null>(null);
