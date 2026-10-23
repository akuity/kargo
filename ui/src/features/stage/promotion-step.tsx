import { faCog, faLinesLeaning, faTerminal } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex, Segmented, Tag, Typography } from 'antd';
import { SegmentedLabeledOption } from 'antd/es/segmented';
import classNames from 'classnames';
import { useMemo, useState } from 'react';

import { useExtensionsContext } from '@ui/extensions/extensions-context';
import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import {
  getStepErrorCount,
  PromotionDirectiveStepStatus
} from '@ui/features/common/promotion-directive-step-status/utils';
import { usePromotionDirectivesRegistryContext } from '@ui/features/promotion-directives/registry/context/use-registry-context';
import { Runner } from '@ui/features/promotion-directives/registry/types';
import { Promotion, PromotionStep } from '@ui/gen/api/v2/models';
import uiPlugins from '@ui/plugins';
import { UiPluginHoles } from '@ui/plugins/atoms/ui-plugin-hole/ui-plugin-holes';

import { StepDuration } from './promotion-step-duration';
import { StepLogs } from './promotion-step-logs';
import { StepStatusIcon } from './promotion-step-status-icon';
import { StepTitle } from './promotion-step-title';
import { objectToYAML } from './utils/promotion';
import { getUserStepAlias } from './utils/promotion-step-groups';
import { getStepLogs } from './utils/step-logs';

export const Step = ({
  step,
  result,
  output,
  promotion,
  stepIndex
}: {
  step: PromotionStep;
  result: PromotionDirectiveStepStatus;
  output?: object;
  promotion?: Promotion;
  stepIndex: number;
}) => {
  const [showDetails, setShowDetails] = useState(false);

  const { registry } = usePromotionDirectivesRegistryContext();

  const { promotionStepExtensions } = useExtensionsContext();

  const stepExtension = promotionStepExtensions.find((ext) => ext.identifier === step.uses);

  const meta = useMemo(() => {
    const runnerMetadata: Runner = registry.runners.find((r) => r.identifier === step.uses) || {
      identifier: step.uses || 'unknown-step',
      unstable_icons: [],
      config: {}
    };

    let userConfig = '';
    if (step?.config) {
      userConfig = objectToYAML(step?.config);
    }

    return {
      spec: runnerMetadata,
      config: userConfig
    };
  }, [registry, step]);

  const retrying = result === PromotionDirectiveStepStatus.RETRYING;
  const running = result === PromotionDirectiveStepStatus.RUNNING;
  const progressing = running || retrying;

  const attempt = getStepErrorCount(stepIndex, promotion?.status);

  // Console output gets its own panel; the YAML view renders it as an
  // unreadable escaped blob. Output keeps that view alongside.
  const logs = useMemo(() => getStepLogs(output), [output]);

  const opts: SegmentedLabeledOption<string>[] = [];

  if (logs) {
    opts.push({
      label: 'Logs',
      value: 'logs',
      icon: <FontAwesomeIcon icon={faTerminal} className='text-xs' />,
      className: 'p-2'
    });
  }

  if (output) {
    opts.push({
      label: 'Output',
      value: 'output',
      icon: <FontAwesomeIcon icon={faLinesLeaning} className='text-xs' />,
      className: 'p-2'
    });
  }

  if (meta?.config) {
    opts.push({
      label: 'Config',
      value: 'config',
      icon: <FontAwesomeIcon icon={faCog} className='text-xs' />,
      className: 'p-2'
    });
  }

  // Which views exist depends on data that streams in -- a step still running
  // has no output yet -- so the selection is resolved per render rather than
  // frozen at mount. An explicit choice wins for as long as that view is still
  // on offer; otherwise the first one does.
  const [chosenOpt, setChosenOpt] = useState<string>();

  const selectedOpt = opts.find((opt) => opt.value === chosenOpt)?.value ?? opts[0]?.value;

  // Serializing a step's whole output runs on every promotion watch tick
  // otherwise, which is exactly the cost the log panel exists to avoid.
  const yamlView = useMemo(
    () => ({ config: meta?.config, output: objectToYAML(output) }),
    [meta?.config, output]
  );

  const filteredUiPlugins = uiPlugins
    .filter((plugin) =>
      plugin.DeepLinkPlugin?.PromotionStep?.shouldRender({
        step,
        result,
        output: output as Record<string, unknown>
      })
    )
    .map((plugin) => plugin.DeepLinkPlugin?.PromotionStep?.render);

  const userAlias = getUserStepAlias(step.as);

  return {
    className: classNames('', {
      'border-green-500': running,
      'border-yellow-500': retrying,
      'border-gray-200 dark:border-neutral-700': !progressing
    }),
    label: (
      <Flex align='start' className='min-w-0' onClick={() => setShowDetails(!showDetails)}>
        <StepStatusIcon status={result} />
        <Flex vertical gap={4} className='flex-1 min-w-0'>
          <Flex align='center' gap={8}>
            <StepTitle title={meta.spec.identifier} />
            {!!userAlias && (
              <Tag className='text-xs' color='blue' title={step.as}>
                {userAlias}
              </Tag>
            )}
            {retrying && (
              <Tag className='text-xs py-0' color='warning' variant='filled'>
                Retrying &middot; attempt {attempt + 1}
              </Tag>
            )}
          </Flex>
          {!!step.description && (
            <Typography.Text type='secondary' className='text-xs'>
              {step.description}
            </Typography.Text>
          )}
        </Flex>
        <Flex align='center' gap={8}>
          {filteredUiPlugins.length > 0 && (
            <UiPluginHoles.DeepLinks.PromotionStep className='ml-2'>
              {filteredUiPlugins.map(
                (ApplyPlugin, idx) =>
                  ApplyPlugin && (
                    <ApplyPlugin
                      result={result}
                      step={step}
                      output={output as Record<string, unknown>}
                      key={idx}
                    />
                  )
              )}
            </UiPluginHoles.DeepLinks.PromotionStep>
          )}
          <StepDuration promotion={promotion} stepIndex={stepIndex} />
        </Flex>
      </Flex>
    ),
    children: stepExtension ? (
      <stepExtension.component
        step={step}
        result={result}
        output={output as Record<string, unknown>}
        promotion={promotion}
      />
    ) : (
      <>
        {opts.length > 1 && (
          <Segmented
            value={selectedOpt}
            size='small'
            options={opts}
            onChange={setChosenOpt}
            className='mb-2'
          />
        )}
        {selectedOpt === 'logs' && logs ? (
          <StepLogs lines={logs} />
        ) : (
          <YamlEditor
            value={yamlView[selectedOpt as keyof typeof yamlView] ?? ''}
            height='200px'
            disabled
          />
        )}
      </>
    )
  };
};
