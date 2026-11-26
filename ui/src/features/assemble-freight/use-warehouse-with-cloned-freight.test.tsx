import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';

import { WarehouseExpanded } from '@ui/extend/types';
import { ChartDiscoveryResult } from '@ui/gen/api/v2/models';

import { constructFreight } from './construct-freight';
import { useWarehouseWithClonedFreight } from './use-warehouse-with-cloned-freight';

const repoURL = 'oci://registry.example.com/charts/app';
const annotations = { 'example.com/release-notes': 'https://example.com/releases/1.2.3' };

describe('useWarehouseWithClonedFreight', () => {
  it.each([
    { name: 'existing subscription', charts: [{ repoURL, versions: ['1.2.4'] }] },
    { name: 'missing subscription', charts: [] }
  ])('preserves annotations when cloning an aged-out chart: $name', ({ charts }) => {
    const warehouse: WarehouseExpanded = {
      status: { discoveredArtifacts: { charts } }
    };
    let discovered: ChartDiscoveryResult | undefined;
    const Probe = () => {
      const enriched = useWarehouseWithClonedFreight(warehouse, {
        origin: { kind: 'Warehouse', name: 'charts' },
        charts: [{ repoURL, version: '1.2.3', annotations }]
      });
      discovered = enriched.status?.discoveredArtifacts?.charts?.[0];
      return null;
    };
    renderToStaticMarkup(<Probe />);
    expect(discovered?.versions?.[0]).toBe('1.2.3');
    expect(discovered?.annotations?.['1.2.3']).toEqual(annotations);
    expect(
      constructFreight({ chart: { artifact: discovered!, info: '1.2.3' } }, 'charts').charts?.[0]
        .annotations
    ).toEqual(annotations);
    expect(warehouse.status?.discoveredArtifacts?.charts).toEqual(charts);
  });
});
