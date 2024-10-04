import {
  faCheck,
  faCircleNotch,
  faCog,
  faFileLines,
  faLinesLeaning,
  faShoePrints,
  faTimes
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Collapse, Flex, Modal, Segmented, Tabs } from 'antd';
import Alert from 'antd/es/alert/Alert';
import { SegmentedOptions } from 'antd/es/segmented';
import classNames from 'classnames';
import { useMemo, useState } from 'react';

import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import { ManifestPreview } from '@ui/features/common/manifest-preview';
import { ModalProps } from '@ui/features/common/modal/use-modal';
import {
  getPromotionDirectiveStepStatus,
  PromotionDirectiveStepStatus
} from '@ui/features/common/promotion-directive-step-status/utils';
import { usePromotionDirectivesRegistryContext } from '@ui/features/promotion-directives/registry/context/use-registry-context';
import { Runner } from '@ui/features/promotion-directives/registry/types';
import { Promotion, PromotionStep } from '@ui/gen/v1alpha1/generated_pb';
import { decodeRawData } from '@ui/utils/decode-raw-data';

const Step = ({
  step,
  result,
  logs
}: {
  step: PromotionStep;
  result: PromotionDirectiveStepStatus;
  logs?: object;
}) => {
  const [showDetails, setShowDetails] = useState(false);

  const { registry } = usePromotionDirectivesRegistryContext();

  const meta = useMemo(() => {
    const runnerMetadata: Runner = registry.runners.find((r) => r.identifier === step.uses) || {
      identifier: step.uses || 'unknown-step',
      unstable_icons: []
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

  if (logs) {
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
    output: logs ? JSON.stringify(logs || {}, null, ' ') : ''
  };

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
        <Flex className={'font-semibold text-base w-full'} align='center'>
          {meta.spec.identifier}
          <Flex className='ml-auto' align='center'>
            <Flex
              align='center'
              className='bg-gray-500 text-white uppercase p-2 rounded-md font-medium mr-3 gap-2 text-sm'
            >
              {meta.spec.unstable_icons.map((icon, i) => (
                <FontAwesomeIcon key={i} icon={icon} />
              ))}
            </Flex>
          </Flex>
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

export const PromotionDetailsModal = ({
  promotion,
  hide,
  visible
}: {
  promotion: Promotion;
} & ModalProps) => {
  const logsByStepAlias: Record<string, object> = useMemo(() => {
    if (promotion?.status?.state?.raw) {
      try {
        const raw = decodeRawData({ result: { case: 'raw', value: promotion.status.state.raw } });

        return JSON.parse(raw);
      } catch (e) {
        // eslint-disable-next-line no-console
        console.error(e);
      }
    }

    return {};
  }, [promotion]);

  return (
    <Modal
      title='Promotion Details'
      open={visible}
      width='800px'
      okText='Close'
      onOk={hide}
      onCancel={hide}
      cancelButtonProps={{ hidden: true }}
    >
      <Tabs defaultActiveKey='1'>
        {promotion.spec?.steps && (
          <Tabs.TabPane tab='Steps' key='1' icon={<FontAwesomeIcon icon={faShoePrints} />}>
            <Collapse
              expandIconPosition='end'
              bordered={false}
              items={promotion.spec.steps.map((step, i) => {
                return Step({
                  step,
                  result: getPromotionDirectiveStepStatus(i, promotion.status),
                  logs: logsByStepAlias?.[step?.as || '']
                });
              })}
            />
            {!!promotion?.status?.message && (
              <Alert message={promotion.status.message} type='error' className='mt4' />
            )}
          </Tabs.TabPane>
        )}
        <Tabs.TabPane tab='YAML' key='2' icon={<FontAwesomeIcon icon={faFileLines} />}>
          <ManifestPreview object={promotion} height='500px' />
        </Tabs.TabPane>
      </Tabs>
    </Modal>
  );
};
