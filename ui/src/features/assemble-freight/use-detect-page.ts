import { useEffect, useState } from 'react';

export const useDetectPage = <T>(
  items: T[],
  selected: T | undefined,
  isEqual: (item: T, selected: T) => boolean,
  pause?: boolean,
  pageLimit = 10
) => {
  const [page, setPage] = useState(1);

  useEffect(() => {
    if (pause || selected === undefined) {
      return;
    }

    const index = items.findIndex((item) => isEqual(item, selected));

    if (index === -1) {
      return;
    }

    setPage(Math.ceil((index + 1) / pageLimit));
  }, [items, selected, pause]);

  return [page, setPage] as const;
};
