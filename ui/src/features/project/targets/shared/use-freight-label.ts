import { getAlias } from '@ui/features/common/utils';
import { useGetFreightMap } from '@ui/features/stage/tabs/freight-history/use-get-freight-map';

// useFreightLabel names a piece of Freight in the few characters a table cell
// or a card footer has room for: the alias someone gave it, or the start of its
// name. One hook, so every surface on the page names it the same way.
export const useFreightLabel = (project: string) => {
  const freightMap = useGetFreightMap(project);
  return (name: string) => getAlias(freightMap[name]) || name.slice(0, 7);
};
