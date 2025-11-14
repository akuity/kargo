import { IconDefinition, faCertificate, faCircleCheck } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useContext } from 'react';

import { ColorContext } from '@ui/context/colors';
import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

const StageIndicator = ({ stageName, icon }: { stageName: string; icon?: IconDefinition }) => {
  const { stageColorMap } = useContext(ColorContext);
  return (
    <div
      className='rounded-md px-3 py-1 mr-2 text-white font-medium text-sm'
      style={{ backgroundColor: stageColorMap[stageName] }}
    >
      {icon && <FontAwesomeIcon icon={icon} className='mr-2' />}
      {stageName}
    </div>
  );
};

const StageStatusList = ({
  title,
  stageNames,
  icon
}: {
  title: string;
  stageNames: string[];
  icon?: IconDefinition;
}) => {
  return (
    <div className='mb-6'>
      <div className='text-xs font-semibold mb-2 uppercase'>{title}</div>
      {stageNames.length > 0 ? (
        <div className='flex items-center gap-2 flex-wrap'>
          {stageNames.map((stageName) => (
            <StageIndicator stageName={stageName} key={stageName} icon={icon} />
          ))}
        </div>
      ) : (
        <div className='w-full bg-gray-100 px-3 py-2 font-medium text-gray-600 rounded'>
          This freight has not been {title} any stages yet.
        </div>
      )}
    </div>
  );
};

export const FreightStatusList = ({ freight }: { freight?: Freight }) => (
  <div>
    <StageStatusList
      title='verified in'
      stageNames={Object.keys(freight?.status?.verifiedIn || {}) || []}
      icon={faCertificate}
    />
    <StageStatusList
      title='approved for'
      stageNames={Object.keys(freight?.status?.approvedFor || {}) || []}
      icon={faCircleCheck}
    />
  </div>
);
