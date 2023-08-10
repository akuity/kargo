import { useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';

export const authTokenKey = 'auth_token';

export const useAuth = () => {
  const navigate = useNavigate();

  const onLogin = (token: string) => {
    localStorage.setItem(authTokenKey, token);
    navigate(paths.home, { replace: true });
  };

  return {
    onLogin
  };
};
