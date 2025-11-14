import { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';

import { FreightTimelineControllerContextType } from '@ui/features/project/pipelines/context/freight-timeline-controller-context';
import { timerangeTypes } from '@ui/features/project/pipelines/freight/filter-timerange-utils';

import { getFreightTimelineFiltersLocalStorage } from '../use-persist-filters';

export const useFreightTimelineControllerStore = (project: string) => {
  const [searchParams, setSearchParams] = useSearchParams();

  const filters = useMemo(() => {
    const filters: FreightTimelineControllerContextType['preferredFilter'] = {
      showAlias: true,
      sources: [],
      timerange: 'all-time',
      showColors: true,
      warehouses: [],
      hideUnusedFreights: false,
      stackedNodesParents: [],
      hideSubscriptions: {},
      images: false,
      view: 'graph',
      showMinimap: true
    };

    if (searchParams.size === 0) {
      return { ...filters, ...getFreightTimelineFiltersLocalStorage(project) };
    }

    const viewParam = searchParams.get('view');
    if (viewParam && viewParam !== '' && ['graph', 'list'].includes(viewParam)) {
      filters.view = viewParam as 'graph' | 'list';
    }

    const showAliasParam = searchParams.get('showAlias');
    if (showAliasParam && showAliasParam !== '') {
      filters.showAlias = showAliasParam === 'true';
    }

    const showMinimap = searchParams.get('showMinimap');
    if (showMinimap && showMinimap !== '') {
      filters.showMinimap = showMinimap === 'true';
    }

    const sourcesParam = searchParams.getAll('sources');
    if (sourcesParam && sourcesParam.length > 0) {
      filters.sources = sourcesParam;
    } else {
      filters.sources = [];
    }

    const timerangeParam = searchParams.get('timerange');
    if (timerangeParam && timerangeParam !== '') {
      filters.timerange = timerangeParam as timerangeTypes;
    }

    const showColorsParam = searchParams.get('showColors');
    if (showColorsParam && showColorsParam !== '') {
      filters.showColors = showColorsParam === 'true';
    }

    const warehousesParam = searchParams.getAll('warehouses');
    if (warehousesParam && warehousesParam.length > 0) {
      filters.warehouses = warehousesParam;
    } else {
      filters.warehouses = [];
    }

    const hideUnusedFreightsParam = searchParams.get('hideUnusedFreights');
    if (hideUnusedFreightsParam && hideUnusedFreightsParam !== '') {
      filters.hideUnusedFreights = hideUnusedFreightsParam === 'true';
    }

    const stackedNodesParentsParam = searchParams.getAll('stackedNodesParents');
    if (stackedNodesParentsParam && stackedNodesParentsParam.length > 0) {
      filters.stackedNodesParents = stackedNodesParentsParam;
    } else {
      filters.stackedNodesParents = [];
    }

    const imagesParam = searchParams.get('images');
    if (imagesParam && imagesParam !== '') {
      filters.images = imagesParam === 'true';
    }

    const hideSubscriptionsParam = searchParams.getAll('hideSubscriptions');
    if (hideSubscriptionsParam && hideSubscriptionsParam.length > 0) {
      for (const hideSubscriptionOf of hideSubscriptionsParam) {
        filters.hideSubscriptions[hideSubscriptionOf] = true;
      }
    } else {
      filters.hideSubscriptions = {};
    }

    return { ...getFreightTimelineFiltersLocalStorage(project), ...filters };
  }, [searchParams]);

  return [
    filters,
    useCallback(
      (nextPartial: Partial<FreightTimelineControllerContextType['preferredFilter']>) => {
        localStorage.setItem(`filters-${project}`, JSON.stringify({ ...filters, ...nextPartial }));

        const currentSearchParams = new URLSearchParams(searchParams);

        currentSearchParams.set('view', `${nextPartial.view}`);

        currentSearchParams.set('showMinimap', `${nextPartial.showMinimap}`);

        currentSearchParams.set('showColors', `${nextPartial.showColors}`);

        currentSearchParams.set('showAlias', `${nextPartial.showAlias}`);

        currentSearchParams.delete('sources');
        if (nextPartial.sources && nextPartial.sources.length > 0) {
          for (const source of nextPartial.sources) {
            currentSearchParams.append('sources', source);
          }
        }

        currentSearchParams.set('timerange', `${nextPartial.timerange}`);

        currentSearchParams.delete('warehouses');
        if (nextPartial.warehouses && nextPartial.warehouses.length > 0) {
          for (const warehouse of nextPartial.warehouses) {
            currentSearchParams.append('warehouses', warehouse);
          }
        }

        currentSearchParams.set('hideUnusedFreights', `${nextPartial.hideUnusedFreights}`);

        currentSearchParams.delete('stackedNodesParents');
        if (nextPartial.stackedNodesParents && nextPartial.stackedNodesParents.length > 0) {
          for (const stackedNodesParent of nextPartial.stackedNodesParents) {
            currentSearchParams.append('stackedNodesParents', stackedNodesParent);
          }
        }

        currentSearchParams.set('images', `${nextPartial.images}`);

        const hideSubscriptionEntries = Object.entries(nextPartial.hideSubscriptions || {});
        currentSearchParams.delete('hideSubscriptions');
        if (hideSubscriptionEntries.length > 0) {
          for (const [name, hideSubscription] of hideSubscriptionEntries) {
            if (hideSubscription) {
              currentSearchParams.append('hideSubscriptions', name);
            }
          }
        }

        setSearchParams(currentSearchParams);
      },
      [searchParams, setSearchParams]
    )
  ] as const;
};
