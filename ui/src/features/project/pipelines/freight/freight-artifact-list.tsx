import { theme, Tooltip, Typography } from 'antd';
import { useMemo } from 'react';

import { RepoSubscription } from '@ui/extend/types';
import { Freight, FreightReference } from '@ui/gen/api/v2/models';

import { FreightArtifact } from './freight-artifact';
import { getChangedArtifacts } from './freight-artifact-changes-utils';
import {
  DEFAULT_MAX_ARTIFACTS,
  FreightArtifactItem,
  getFreightArtifacts
} from './freight-artifact-list-utils';
import { getSubscriptionLabel } from './subscription-label-utils';

type FreightArtifactListProps = {
  freight?: Freight | FreightReference;
  // max is the number of artifacts to render before collapsing the rest into a
  // "+N more" indicator.
  max?: number;
  // The piece of freight its Warehouse discovered before this one. When set,
  // artifacts the same as in it are muted, so the ones that differ stand out.
  previousFreight?: Freight | FreightReference;
  // whether each tag names the Warehouse subscription its artifact came from
  showSubscriptionNames?: boolean;
  // the subscriptions of the freight's Warehouse, to name the ones its
  // artifacts do not record
  subscriptions?: RepoSubscription[];
};

// FreightArtifactList renders up to `max` artifact tags for a piece of Freight,
// collapsing any overflow into a "+N more" indicator that lists the hidden
// artifacts on hover. Shared by the freight timeline card and the Stage
// drawer's current-freight panel. It renders a flat fragment so callers control
// the surrounding layout.
export const FreightArtifactList = ({
  freight,
  max = DEFAULT_MAX_ARTIFACTS,
  previousFreight,
  showSubscriptionNames,
  subscriptions
}: FreightArtifactListProps) => {
  const { token } = theme.useToken();

  const artifacts = getFreightArtifacts(freight);
  const overflow = artifacts.length - max;

  const changedArtifacts = useMemo(
    () => (previousFreight ? getChangedArtifacts(freight, previousFreight) : undefined),
    [freight, previousFreight]
  );

  const subscriptionLabel = (artifact: FreightArtifactItem) =>
    showSubscriptionNames ? getSubscriptionLabel(artifact, subscriptions) : undefined;

  return (
    <>
      {artifacts.slice(0, max).map((artifact, i) => (
        <FreightArtifact
          key={i}
          artifact={artifact}
          changed={changedArtifacts?.has(artifact)}
          subscriptionLabel={subscriptionLabel(artifact)}
        />
      ))}
      {overflow > 0 && (
        <Tooltip
          // The surface popovers and dropdowns use, rather than the tooltip's
          // own: white on the light theme, where the near-black tooltip makes
          // the blue tags read badly, and a step above the cards on the dark.
          color={token.colorBgElevated}
          title={
            // a freight with dozens of artifacts scrolls rather than running
            // the tooltip off the screen
            <div className='flex flex-col gap-[2px] max-h-[60vh] overflow-y-auto'>
              {artifacts.slice(max).map((artifact, i) => (
                <FreightArtifact
                  key={i}
                  artifact={artifact}
                  changed={changedArtifacts?.has(artifact)}
                  subscriptionLabel={subscriptionLabel(artifact)}
                />
              ))}
            </div>
          }
        >
          <Typography.Text type='secondary' className='text-[10px] cursor-default'>
            +{overflow} more
          </Typography.Text>
        </Tooltip>
      )}
    </>
  );
};
