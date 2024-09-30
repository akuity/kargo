import {
  faCaretDown,
  faCaretUp,
  faCheck,
  faCircleNotch,
  faFileLines,
  faShoePrints,
  faTimes
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex, Modal, Tabs } from 'antd';
import Alert from 'antd/es/alert/Alert';
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

const Step = ({ step, result }: { step: PromotionStep; result: PromotionDirectiveStepStatus }) => {
  const [showDetails, setShowDetails] = useState(false);

  const { registry } = usePromotionDirectivesRegistryContext();

  const meta = useMemo(() => {
    const runnerMetadata: Runner = registry.runners.find((r) => r.identifier === step.uses) || {
      identifier: step.uses || 'unknown-step',
      unstable_icons: []
    };

    const userConfig = JSON.stringify(
      JSON.parse(
        decodeRawData({
          result: { case: 'raw', value: step?.config?.raw || new Uint8Array() }
        })
      ),
      null,
      ' '
    );

    return {
      spec: runnerMetadata,
      config: userConfig
    };
  }, [registry, step]);

  const progressing = result === PromotionDirectiveStepStatus.RUNNING;
  const success = result === PromotionDirectiveStepStatus.SUCCESS;
  const failed = result === PromotionDirectiveStepStatus.FAILED;

  return (
    <Flex
      className={classNames('rounded-md border-2 border-solid p-2 mb-3', {
        'border-green-500': progressing,
        'border-gray-200': !progressing
      })}
      vertical
    >
      <Flex align='center'>
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
        <Flex className='font-semibold text-base w-full' align='center'>
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
            {step.config && (
              <FontAwesomeIcon
                icon={showDetails ? faCaretUp : faCaretDown}
                onClick={() => setShowDetails(!showDetails)}
                className='mr-2 text-blue-500 cursor-pointer'
              />
            )}
          </Flex>
        </Flex>
      </Flex>
      {showDetails && <YamlEditor value={meta.config} height='200px' className='mt-2' disabled />}
    </Flex>
  );
};

export const PromotionDetailsModal = ({
  promotion,
  hide,
  visible
}: {
  promotion: Promotion;
} & ModalProps) => {
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
            {promotion.spec.steps.map((step, i) => (
              <Step
                key={i}
                step={step}
                result={getPromotionDirectiveStepStatus(i, promotion.status)}
              />
            ))}
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
