import {
  faCheck,
  faCircleNotch,
  faCog,
  faLinesLeaning,
  faTimes
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex, Segmented, Tag } from 'antd';
import { SegmentedOptions } from 'antd/es/segmented';
import classNames from 'classnames';
import { useMemo, useState } from 'react';

import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import { PromotionDirectiveStepStatus } from '@ui/features/common/promotion-directive-step-status/utils';
import { usePromotionDirectivesRegistryContext } from '@ui/features/promotion-directives/registry/context/use-registry-context';
import { Runner } from '@ui/features/promotion-directives/registry/types';
import { PromotionStep } from '@ui/gen/api/v1alpha1/generated_pb';
import uiPlugins from '@ui/plugins';
import { UiPluginHoles } from '@ui/plugins/atoms/ui-plugin-hole/ui-plugin-holes';
import { decodeRawData } from '@ui/utils/decode-raw-data';

export const Step = ({
  step,
  result,
  output
}: {
  step: PromotionStep;
  result: PromotionDirectiveStepStatus;
  output?: object;
}) => {
  const [showDetails, setShowDetails] = useState(false);

  const { registry } = usePromotionDirectivesRegistryContext();

  const meta = useMemo(() => {
    const runnerMetadata: Runner = registry.runners.find((r) => r.identifier === step.uses) || {
      identifier: step.uses || 'unknown-step',
      unstable_icons: [],
      config: {}
    };

    let userConfig = '';
    if (step?.config?.raw) {
      userConfig = JSON.stringify(
        JSON.parse(
          decodeRawData({
            result: { case: 'raw', value: step?.config?.raw || new Uint8Array() }
          })
        ),
        null,
        ' '
      );
    }

    return {
      spec: runnerMetadata,
      config: userConfig
    };
  }, [registry, step]);

  const progressing = result === PromotionDirectiveStepStatus.RUNNING;
  const success = result === PromotionDirectiveStepStatus.SUCCESS;
  const failed = result === PromotionDirectiveStepStatus.FAILED;

  const opts: SegmentedOptions<string> = [];

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

  const [selectedOpts, setSelectedOpts] = useState(
    // @ts-expect-error value is there
    opts?.[0]?.value
  );

  const yamlView = {
    config: meta?.config,
    output: output ? JSON.stringify(output || {}, null, ' ') : ''
  };

  const filteredUiPlugins = uiPlugins
    .filter((plugin) =>
      plugin.DeepLinkPlugin?.PromotionStep?.shouldRender({
        step,
        result,
        output: output as Record<string, unknown>
      })
    )
    .map((plugin) => plugin.DeepLinkPlugin?.PromotionStep?.render);

  return {
    className: classNames('', {
      'border-green-500': progressing,
      'border-gray-200': !progressing
    }),
    label: (
      <Flex align='center' onClick={() => setShowDetails(!showDetails)}>
        <Flex
          align='center'
          justify='center'
          className='mr-2'
          style={{ width: '20px', height: '20px', marginBottom: '1px' }}
        >
          {progressing && <FontAwesomeIcon spin icon={faCircleNotch} />}
          {success && <FontAwesomeIcon icon={faCheck} className='text-green-500' />}
          {failed && <FontAwesomeIcon icon={faTimes} className='text-red-500' />}
        </Flex>
        <Flex className={'w-full'} align='center'>
          <span className='font-semibold text-base '>{meta.spec.identifier}</span>
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
          {!!step?.as && (
            <Tag className='text-xs ml-auto mr-5' color='blue'>
              {step.as}
            </Tag>
          )}
        </Flex>
      </Flex>
    ),
    children: (
      <>
        {opts.length > 1 && (
          <Segmented
            value={selectedOpts}
            size='small'
            options={opts}
            onChange={setSelectedOpts}
            className='mb-2'
          />
        )}
        <YamlEditor
          value={yamlView[selectedOpts as keyof typeof yamlView]}
          height='200px'
          disabled
        />
      </>
    )
  };
};
