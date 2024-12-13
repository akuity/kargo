import { useMutation } from '@connectrpc/connect-query';
import {
  faCheck,
  faCircleNotch,
  faCog,
  faFileLines,
  faLinesLeaning,
  faShoePrints,
  faStopCircle,
  faTimes
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Collapse, Descriptions, Flex, message, Modal, Segmented, Tabs, Tag } from 'antd';
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
import { canAbortPromotion } from '@ui/features/stage/utils/promotion';
import { abortPromotion } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Promotion, PromotionStep } from '@ui/gen/v1alpha1/generated_pb';
import uiPlugins from '@ui/plugins';
import { UiPluginHoles } from '@ui/plugins/ui-plugin-hole/ui-plugin-holes';
import { decodeRawData } from '@ui/utils/decode-raw-data';

const Step = ({
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

  const UiPlugins = uiPlugins
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
          {UiPlugins.length > 0 && (
            <UiPluginHoles.DeepLinks.PromotionStep className='ml-2'>
              {UiPlugins.map(
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

export const PromotionDetailsModal = ({
  promotion,
  hide,
  visible,
  project
}: {
  promotion: Promotion;
  project: string;
} & ModalProps) => {
  const abortPromotionMutation = useMutation(abortPromotion, {
    onSuccess: () =>
      // Abort promotion annotates the Promotion resource and then controller acts
      message.success({
        content: `Abort Promotion ${promotion.metadata?.name} requested successfully.`
      })
  });

  const outputsByStepAlias: Record<string, object> = useMemo(() => {
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

  const confirmAbortRequest = () =>
    Modal.confirm({
      width: '656px',
      icon: <FontAwesomeIcon icon={faStopCircle} className='text-lg text-red-500 mr-5' />,
      title: 'Abort Promotion Request',
      onOk: () => abortPromotionMutation.mutate({ project, name: promotion?.metadata?.name }),
      okText: 'Abort',
      okButtonProps: {
        danger: true
      },
      content: (
        <Descriptions
          size='small'
          className='mt-2'
          column={1}
          bordered
          items={[
            {
              key: 'name',
              label: 'Name',
              children: promotion.metadata?.name
            },
            {
              key: 'date',
              label: 'Start Date',
              children: promotion.metadata?.creationTimestamp?.toDate().toString()
            }
          ]}
        />
      )
    });

  return (
    <Modal
      title='Promotion Details'
      open={visible}
      width='800px'
      okButtonProps={{ hidden: true }}
      onOk={hide}
      onCancel={hide}
    >
      <Tabs
        defaultActiveKey='1'
        tabBarExtraContent={
          canAbortPromotion(promotion) && (
            <Button
              danger
              icon={<FontAwesomeIcon icon={faStopCircle} className='text-sm' />}
              onClick={confirmAbortRequest}
              size='small'
            >
              Abort
            </Button>
          )
        }
      >
        {promotion.spec?.steps && (
          <Tabs.TabPane tab='Steps' key='1' icon={<FontAwesomeIcon icon={faShoePrints} />}>
            <Collapse
              expandIconPosition='end'
              bordered={false}
              items={promotion.spec.steps.map((step, i) => {
                return Step({
                  step,
                  result: getPromotionDirectiveStepStatus(i, promotion.status),
                  output: outputsByStepAlias?.[step?.as || '']
                });
              })}
            />
            {!!promotion?.status?.message && (
              <Alert message={promotion.status.message} type='error' className='mt-4' />
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
