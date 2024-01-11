import React from 'react';

export interface AuthContextType {
  isLoggedIn: boolean;
  login: (token: string, authToken?: string) => void;
  logout: () => void;
}

export const AuthContext = React.createContext<AuthContextType | null>(null);
