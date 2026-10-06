import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex, Tooltip, Typography } from 'antd';

import { Freight, FreightReference } from '@ui/gen/api/v2/models';

import { FreightArtifact } from './freight-artifact';
import {
  DEFAULT_MAX_ARTIFACTS,
  getFreightArtifacts,
  getHiddenFreightArtifactCounts
} from './freight-artifact-list-utils';

type FreightArtifactListProps = {
  freight?: Freight | FreightReference;
  // max is the number of artifacts to render before collapsing the rest into a
  // "+N more" indicator.
  max?: number;
  expand?: boolean;
};

// FreightArtifactList renders up to `max` artifact tags for a piece of Freight,
// collapsing any overflow into a "+N more" indicator that reveals a per-kind
// breakdown of the hidden artifacts on hover. Shared by the freight timeline
// card and the Stage drawer's current-freight panel. It renders a flat fragment
// so callers control the surrounding layout.
export const FreightArtifactList = ({
  freight,
  max = DEFAULT_MAX_ARTIFACTS,
  expand
}: FreightArtifactListProps) => {
  const artifacts = getFreightArtifacts(freight);
  const overflow = artifacts.length - max;

  return (
    <>
      {artifacts.slice(0, max).map((artifact, i) => (
        <FreightArtifact key={i} artifact={artifact} expand={expand} />
      ))}
      {overflow > 0 && (
        <Tooltip
          title={
            <Flex vertical align='center'>
              {getHiddenFreightArtifactCounts(freight, max).map(({ icon, count }) => (
                <div key={icon.iconName} className='text-[10px]'>
                  {count}x <FontAwesomeIcon icon={icon} />
                </div>
              ))}
            </Flex>
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
