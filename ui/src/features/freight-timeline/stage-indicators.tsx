import { Tooltip } from 'antd';
import { useContext } from 'react';

import { ColorContext } from '@ui/context/colors';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

const StageIndicator = ({
  stage,
  backgroundColor,
  faded
}: {
  stage: Stage;
  backgroundColor: string;
  faded?: boolean;
}) => {
  return (
    <Tooltip title={stage ? stage.metadata?.name : null} placement='right'>
      <div
        className={`my-1 flex-shrink h-full flex items-center justify-center flex-col w-full rounded`}
        style={{
          background: faded
            ? '#333'
            : `linear-gradient(60deg,rgb(255 255 255/0%),rgb(200 200 200/30%)), ${backgroundColor}`
        }}
      />
    </Tooltip>
  );
};

export const StageIndicators = (props: { stages: Stage[]; faded?: boolean }) => {
  const { stageColorMap } = useContext(ColorContext);
  return (props.stages || []).length > 0 ? (
    <div
      className={`flex flex-col align-center h-full justify-center flex-shrink mr-2`}
      style={{ width: '20px' }}
    >
      {(props.stages || []).map((s) => (
        <StageIndicator
          stage={s}
          backgroundColor={stageColorMap[s?.metadata?.name || '']}
          key={s?.metadata?.uid}
          faded={props.faded}
        />
      ))}
    </div>
  ) : (
    <></>
  );
};
