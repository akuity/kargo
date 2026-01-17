import { useParams } from 'react-router-dom';

import { ConfigMaps } from '@ui/features/common/settings/config-maps/config-maps';

export const ConfigMapsSettings = () => {
  const { name: project } = useParams();

  return <ConfigMaps project={project} />;
};
