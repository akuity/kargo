import { formatDistance } from 'date-fns';
import { useMemo } from 'react';

import { Freight } from '@ui/gen/api/v2/models';

export const useGetFreightCreation = (freight?: Freight) =>
  useMemo(() => {
    if (!freight?.metadata?.creationTimestamp) {
      return {
        relative: '',
        abs: null
      };
    }

    const creationDate = new Date(freight?.metadata?.creationTimestamp);

    return {
      relative: formatDistance(creationDate, new Date(), { addSuffix: false })?.replace(
        'about',
        ''
      ),
      abs: creationDate
    };
  }, [freight]);
