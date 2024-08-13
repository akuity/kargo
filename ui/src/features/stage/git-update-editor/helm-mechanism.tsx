import { HelmCharts } from './helm-charts';
import { HelmImages } from './helm-images';

export const HelmMechanism = () => {
  return (
    <>
      <HelmImages />
      <HelmCharts />
    </>
  );
};
