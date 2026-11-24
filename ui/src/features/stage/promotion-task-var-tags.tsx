import { Flex, Popover, Tag } from 'antd';

import { ExpressionVariable } from '@ui/gen/api/v2/models';

const maxVisibleVars = 3;

const VarTag = ({ variable }: { variable: ExpressionVariable }) => {
  const text = `${variable.name}: ${variable.value}`;

  return (
    <Tag className='text-xs max-w-full truncate' title={text}>
      {text}
    </Tag>
  );
};

export const TaskVarTags = ({ vars }: { vars: ExpressionVariable[] }) => {
  const visibleVars = vars.slice(0, maxVisibleVars);
  const hiddenVars = vars.slice(maxVisibleVars);

  return (
    <Flex wrap gap={4} className='min-w-0'>
      {visibleVars.map((variable) => (
        <VarTag key={variable.name} variable={variable} />
      ))}
      {hiddenVars.length > 0 && (
        <Popover
          content={
            <Flex vertical gap={4} align='start' className='max-w-md'>
              {hiddenVars.map((variable) => (
                <VarTag key={variable.name} variable={variable} />
              ))}
            </Flex>
          }
        >
          <Tag className='text-xs'>+{hiddenVars.length} more</Tag>
        </Popover>
      )}
    </Flex>
  );
};
