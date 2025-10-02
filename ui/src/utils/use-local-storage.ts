import { useState } from 'react';

export const useLocalStorage = <T>(key: string, initialValue?: T) => {
  const [storedValue, _setStoredValue] = useState(() => {
    try {
      const item = window.localStorage.getItem(key);
      return item ? JSON.parse(item) : initialValue;
    } catch (_) {
      return initialValue;
    }
  });

  const setStoredValue: typeof _setStoredValue = (storedValue) => {
    _setStoredValue(storedValue);

    if (!storedValue) {
      window.localStorage.removeItem(key);
      return;
    }
    window.localStorage.setItem(key, JSON.stringify(storedValue));
  };

  return [storedValue as T, setStoredValue as React.Dispatch<T>] as const;
};
