import { useEffect, useState } from 'react';

export function useDocumentEvent<T>(event: string, callback: () => T) {
  const [value, setValue] = useState<T>(callback());

  useEffect(() => {
    const handler = () => setValue(callback());
    document.addEventListener(event, handler);
    return () => document.removeEventListener(event, handler);
  }, [event]);

  return value;
}
