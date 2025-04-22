import { formatDistance } from 'date-fns';
import { useMemo } from 'react';

import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

export const useGetFreightCreation = (freight: Freight) =>
  useMemo(() => {
    const creationDate = timestampDate(freight?.metadata?.creationTimestamp);

    if (!creationDate) {
      return {
        relative: '',
        abs: creationDate
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
