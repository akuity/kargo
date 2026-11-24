import { formatDistance } from 'date-fns';
import { useMemo } from 'react';

import { Freight } from '@ui/gen/api/v2/models';

import { getFreightDiscoveredAt } from './get-freight-discovered-at';

export const useGetFreightCreation = (freight?: Freight) =>
  useMemo(() => {
    const creationDate = getFreightDiscoveredAt(freight);

    if (!creationDate) {
      return {
        relative: '',
        abs: null
      };
    }

    return {
      relative: formatDistance(creationDate, new Date(), { addSuffix: false })?.replace(
        'about',
        ''
      ),
      abs: creationDate
    };
  }, [freight]);
