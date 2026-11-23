import { useCallback } from 'react';

import { getAlias } from '@ui/features/common/utils';
import { useGetFreightMap } from '@ui/features/stage/tabs/freight-history/use-get-freight-map';
import { FreightReference } from '@ui/gen/api/v2/models';

// useFreightLabel names a piece of Freight in the few characters a table cell
// or a card footer has room for: the alias someone gave it, or the start of its
// name. One hook, so every surface on the page names it the same way.
export const useFreightLabel = (project: string) => {
  const freightMap = useGetFreightMap(project);
  return (name: string) => getAlias(freightMap[name]) || name.slice(0, 7);
};

// useFreightReference resolves a piece of Freight, by name, to the artifacts
// it carries. A Target's status names its Freight and nothing more, and the
// artifacts are what a reader recognises it by.
export const useFreightReference = (project: string) => {
  const freightMap = useGetFreightMap(project);
  return useCallback(
    (name: string): FreightReference | undefined => {
      const freight = freightMap[name];
      return (
        freight && {
          name,
          origin: freight.origin,
          images: freight.images,
          charts: freight.charts,
          commits: freight.commits,
          artifacts: freight.artifacts
        }
      );
    },
    [freightMap]
  );
};
