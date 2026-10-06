import { Flex, Result } from 'antd';

// Positioned like PipelineEmpty -- see its note on why this is not in flow.
export const PipelineError = (props: { message?: string }) => (
  <Flex align='center' justify='center' className='absolute top-16 left-0 right-0 px-4'>
    <Result status='warning' title='Could not load the pipeline' subTitle={props.message} />
  </Flex>
);
