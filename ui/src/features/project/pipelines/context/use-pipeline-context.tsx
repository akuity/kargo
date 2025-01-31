import { useContext } from 'react';

import { PipelineContext } from './pipeline-context';

export const usePipelineContext = () => useContext(PipelineContext);
