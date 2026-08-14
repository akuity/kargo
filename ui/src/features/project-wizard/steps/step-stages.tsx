import {
  faDiagramProject,
  faQuestionCircle,
  faTrash,
  faWandMagicSparkles
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Flex, Popover, Space, Typography } from 'antd';

import { ColorContext } from '@ui/context/colors';
import { usePromotionDirectivesRegistryContext } from '@ui/features/promotion-directives/registry/context/use-registry-context';
import { CreateStageWizard } from '@ui/features/stage/create-stage-wizard';
import { RunnerWithConfiguration } from '@ui/features/stage/promotion-steps-wizard/types';
import { ColorMapHex } from '@ui/features/stage/utils';
import { Stage } from '@ui/gen/api/v2/models';

import { StageDraft, exampleStages, initialStage } from '../types';

// Example stages need at least one promotion step, otherwise a Stage is a
// control-flow Stage (Stage.IsControlFlow() is true when it has no steps). An
// http GET is the neutral choice: it's in the promotion-step registry, needs no
// repo or Argo CD (just outbound network), and succeeds — a normal Stage that
// promotes out of the box, an obvious placeholder to replace with real steps.
// No `as` alias: Kargo reserves the `step-N`/`task-N` pattern and auto-names
// steps, so leave it unset.
const exampleStepSpecs: { uses: string; as?: string; config: Record<string, unknown> }[] = [
  { uses: 'http', config: { method: 'GET', url: 'https://httpbin.org/get' } }
];

type StageCardProps = {
  stage: StageDraft;
  projectName: string;
  warehouses: string[];
  // sibling stages (excluding this one) for upstream Freight selection
  siblingStages: Stage[];
  onChange: (stage: StageDraft) => void;
  onRemove: () => void;
};

const StageCard = ({
  stage,
  projectName,
  warehouses,
  siblingStages,
  onChange,
  onRemove
}: StageCardProps) => (
  <Card
    title={
      <span className='flex items-center gap-2'>
        {stage.color && (
          <span
            className='inline-block w-2.5 h-2.5 rounded'
            style={{ backgroundColor: ColorMapHex[stage.color] ?? stage.color }}
          />
        )}
        <FontAwesomeIcon icon={faDiagramProject} className='text-gray-400' />
        <span className='font-mono text-sm'>{stage.name || '(unnamed)'}</span>
      </span>
    }
    extra={
      <Button
        type='text'
        danger
        size='small'
        icon={<FontAwesomeIcon icon={faTrash} size='sm' />}
        onClick={onRemove}
      />
    }
  >
    {/* The exact form used by the app's Create Stage drawer */}
    <CreateStageWizard
      value={stage}
      onChange={onChange}
      projectName={projectName}
      warehouses={warehouses}
      stages={siblingStages}
    />
  </Card>
);

type StepStagesProps = {
  value: StageDraft[];
  projectName: string;
  warehouses: string[];
  onChange: (value: StageDraft[]) => void;
};

export const StepStages = ({ value, projectName, warehouses, onChange }: StepStagesProps) => {
  const { registry } = usePromotionDirectivesRegistryContext();

  const add = () => onChange([...value, initialStage()]);

  // Resolve the example steps against the registry (fresh instances each call
  // so stages don't share step objects). Missing runners are skipped.
  const exampleSteps = (): RunnerWithConfiguration[] =>
    exampleStepSpecs
      .map((spec): RunnerWithConfiguration | undefined => {
        const runner = registry.runners.find((r) => r.identifier === spec.uses);
        return runner
          ? { ...runner, as: spec.as, state: spec.config as RunnerWithConfiguration['state'] }
          : undefined;
      })
      .filter((r): r is RunnerWithConfiguration => !!r);

  const loadExample = () =>
    onChange([
      ...value,
      ...exampleStages(warehouses[0] || 'guestbook').map((s) => ({ ...s, steps: exampleSteps() }))
    ]);

  const siblingStagesFor = (index: number): Stage[] =>
    value
      .filter((_, x) => x !== index)
      .map((s) => ({ metadata: { name: s.name }, spec: { requestedFreight: s.requestedFreight } }));

  // Reused RequestedFreight renders upstream stages as StageTags that read
  // stageColorMap from ColorContext, which isn't provided outside the pipeline
  // view. Supply one derived from the draft colors so tags render (and color).
  const stageColorMap = Object.fromEntries(
    value.filter((s) => s.name).map((s) => [s.name, (s.color && ColorMapHex[s.color]) || ''])
  );

  return (
    <ColorContext.Provider value={{ stageColorMap, warehouseColorMap: {} }}>
      <Card
        type='inner'
        title={
          <Space size={4}>
            Stages
            <Popover content='A Stage models an environment in your pipeline. It defines what Freight it can receive, where that Freight comes from, and the promotion steps that run when Freight arrives.'>
              <Typography.Text type='secondary'>
                <FontAwesomeIcon icon={faQuestionCircle} size='xs' />
              </Typography.Text>
            </Popover>
          </Space>
        }
        extra={
          <Button icon={<FontAwesomeIcon icon={faDiagramProject} />} onClick={add}>
            Add Stage
          </Button>
        }
      >
        {value.length === 0 ? (
          <div className='flex flex-col items-center py-10 text-center'>
            <FontAwesomeIcon icon={faDiagramProject} className='text-2xl text-gray-300 mb-3' />
            <div className='text-sm text-gray-500 max-w-md mb-5'>
              Add Stages to define your promotion pipeline. Load the example to scaffold a{' '}
              <code>dev → staging → prod</code> chain inspired by akuity&apos;s{' '}
              <code>kargo-simple</code> example. You can also continue and add Stages later.
            </div>
            <Space>
              <Button
                type='primary'
                icon={<FontAwesomeIcon icon={faWandMagicSparkles} />}
                onClick={loadExample}
              >
                Load example
              </Button>
              <Button icon={<FontAwesomeIcon icon={faDiagramProject} />} onClick={add}>
                Add Stage
              </Button>
            </Space>
          </div>
        ) : (
          <Flex gap={16} vertical>
            {value.map((stage, i) => (
              <StageCard
                key={i}
                stage={stage}
                projectName={projectName}
                warehouses={warehouses}
                siblingStages={siblingStagesFor(i)}
                onChange={(next) => onChange(value.map((s, x) => (x === i ? next : s)))}
                onRemove={() => onChange(value.filter((_, x) => x !== i))}
              />
            ))}
          </Flex>
        )}
      </Card>
    </ColorContext.Provider>
  );
};
