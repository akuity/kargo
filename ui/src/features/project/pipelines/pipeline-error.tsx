import { Flex, Result } from 'antd';

/**
 * Shown in place of the pipeline when the Warehouse or Stage list could not be
 * read. Kept distinct from PipelineEmpty: both leave nothing to draw, but only
 * one of them means the project is actually empty.
 *
 * Positioned like PipelineEmpty -- see its note on why this is not in flow.
 */
export const PipelineError = (props: { message?: string }) => (
  <Flex align='center' justify='center' className='absolute top-16 left-0 right-0 px-4'>
    <Result status='warning' title='Could not load the pipeline' subTitle={props.message} />
  </Flex>
);
