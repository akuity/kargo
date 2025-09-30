import { useEffect, useState } from 'react';

import { DiscoveredCommit, DiscoveredImageReference } from '@ui/gen/api/v1alpha1/generated_pb';

export const useDetectPage = (
  items: DiscoveredImageReference[] | DiscoveredCommit[] | string[],
  selected?: DiscoveredImageReference | DiscoveredCommit | string,
  pause?: boolean,
  pageLimit = 10
) => {
  const [page, setPage] = useState(1);

  useEffect(() => {
    if (pause) {
      return;
    }

    const index = items.findIndex((item) => {
      if (typeof item === 'string') {
        return item === selected;
      }

      if (item.$typeName === 'github.com.akuity.kargo.api.v1alpha1.DiscoveredCommit') {
        const _selected = selected as DiscoveredCommit;
        return item?.id === _selected?.id && item?.tag === _selected?.tag;
      }

      if (item.$typeName === 'github.com.akuity.kargo.api.v1alpha1.DiscoveredImageReference') {
        const _selected = selected as DiscoveredImageReference;
        return item?.tag === _selected?.tag;
      }
    });

    if (index === -1) {
      return;
    }

    setPage(Math.ceil((index + 1) / pageLimit));
  }, [items, selected, pause]);

  return [page, setPage] as const;
};
