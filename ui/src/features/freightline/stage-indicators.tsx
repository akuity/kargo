import { Tooltip } from 'antd';
import { useContext } from 'react';

import { ColorContext } from '@ui/context/colors';
import { Stage } from '@ui/gen/v1alpha1/types_pb';

const StageIndicator = (props: { stage: Stage; backgroundColor: string }) => {
  const { stage, backgroundColor } = props;
  return (
    <Tooltip title={stage ? stage.metadata?.name : null} placement='right'>
      <div
        className={`my-1 flex-shrink h-full flex items-center justify-center flex-col w-full rounded`}
        style={{
          background: `linear-gradient(60deg,rgb(255 255 255/0%),rgb(200 200 200/30%)), ${backgroundColor}`
        }}
      />
    </Tooltip>
  );
};

export const StageIndicators = (props: { stages: Stage[] }) => {
  const stageColorMap = useContext(ColorContext);
  return (props.stages || []).length > 0 ? (
    <div
      className={`flex flex-col align-center h-full justify-center w-full flex-grow mr-2`}
      style={{ width: '80px' }}
    >
      {(props.stages || []).map((s) => (
        <StageIndicator
          stage={s}
          backgroundColor={stageColorMap[s?.metadata?.uid || '']}
          key={s?.metadata?.uid}
        />
      ))}
    </div>
  ) : (
    <></>
  );
};
