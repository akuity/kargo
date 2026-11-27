import { QueryClient, useMutation, useQueryClient } from '@tanstack/react-query';

import {
  ApproveFreightMutationVariables,
  getApproveFreightMutationOptions,
  getGetFreightQueryKey,
  getQueryFreightsRestQueryKey
} from '@ui/gen/api/v2/core/core';

type OnApproval = (variables: ApproveFreightMutationVariables) => void;

export const getApprovalMutationOptions = (client: QueryClient, onApproval?: OnApproval) =>
  getApproveFreightMutationOptions({
    mutation: {
      onSuccess: async (_, variables) => {
        const [listPath] = getQueryFreightsRestQueryKey(variables.project);
        const [detailPrefix] = getGetFreightQueryKey(variables.project, '');

        // Refresh filtered lists and open details even if the Freight watch is disconnected.
        await client.invalidateQueries({
          predicate: ({ queryKey: [path] }) =>
            typeof path === 'string' && (path === listPath || path.startsWith(detailPrefix))
        });

        onApproval?.(variables);
      }
    }
  });

export const useApproveFreight = (onApproval?: OnApproval) =>
  useMutation(getApprovalMutationOptions(useQueryClient(), onApproval));
