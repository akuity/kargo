import { faDocker, faGitAlt } from '@fortawesome/free-brands-svg-icons';
import {
  faCaretDown,
  faCaretUp,
  faClipboard,
  faClock,
  faClone,
  faCodeCommit,
  faCodePullRequest,
  faDharmachakra,
  faDrawPolygon,
  faFileLines,
  faHammer,
  faHeart,
  faPenNib,
  faShoePrints,
  faTextSlash,
  faTurnUp,
  IconDefinition
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex, Modal, Tabs } from 'antd';
import classNames from 'classnames';
import { useState } from 'react';

import { Promotion, PromotionSpec } from '@ui/gen/v1alpha1/generated_pb';

import YamlEditor from '../common/code-editor/yaml-editor-lazy';
import { ManifestPreview } from '../common/manifest-preview';
import { ModalProps } from '../common/modal/use-modal';

import { testDirectives } from './test-directives';

const builtInDirectives: { [key: string]: IconDefinition[] } = {
  'git-clone': [faGitAlt, faClone],
  'kargo-render': [faDrawPolygon],
  'kustomize-set-image': [faDocker, faPenNib],
  'kustomize-build': [faDocker, faHammer],
  'helm-update-image': [faDharmachakra, faDocker],
  'helm-update-chart': [faDharmachakra],
  'helm-template': [faDharmachakra],
  'argocd-health': [faHeart],
  'git-commit': [faGitAlt, faCodeCommit],
  'git-push': [faGitAlt, faTurnUp],
  'pr-open': [faCodePullRequest],
  'git-overwrite': [faGitAlt, faTextSlash],
  'pr-wait': [faCodePullRequest, faClock],
  copy: [faClipboard]
};

// todo: replace any with proper Step type
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const Step = ({ step, order, selected }: { step: any; order: number; selected?: boolean }) => {
  const [showDetails, setShowDetails] = useState(false);

  return (
    <Flex
      className={classNames('rounded-md border-2 border-solid p-2 mb-3', {
        'border-green-500': selected,
        'border-gray-200': !selected
      })}
      vertical
    >
      <Flex align='center'>
        <Flex
          align='center'
          justify='center'
          className='rounded-full p-1 text-white bg-gray-500 font-bold mr-4'
          style={{ width: '20px', height: '20px', marginBottom: '1px' }}
        >
          {order + 1}
        </Flex>
        <Flex className='font-semibold text-base w-full' align='center'>
          {step.step}
          <Flex className='ml-auto' align='center'>
            <Flex
              align='center'
              className='bg-gray-500 text-white uppercase p-2 rounded-md font-medium mr-3 gap-2 text-sm'
            >
              {builtInDirectives[step.step] ? (
                <>
                  {builtInDirectives[step.step].map((icon, i) => (
                    <FontAwesomeIcon key={i} icon={icon} />
                  ))}
                </>
              ) : (
                <span className='text-xs'>Custom Directive</span>
              )}
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
      {showDetails && (
        <YamlEditor
          value={JSON.stringify(step.config, null, 2)}
          height='200px'
          className='mt-2'
          disabled
        />
      )}
    </Flex>
  );
};

export const PromotionDetailsModal = ({
  promotion,
  hide,
  visible,
  currentStep
}: {
  // todo: remove type extension once Promotion type is updated
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  promotion: Promotion & { spec: PromotionSpec & { steps: any[] } };
  currentStep?: number; // currentStep should indicate which step the promotion is on
} & ModalProps) => {
  promotion.spec.steps = testDirectives;
  return (
    <Modal
      title='Promotion Details'
      visible={visible}
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
              <Step key={i} step={step} order={i} selected={i === currentStep} />
            ))}
          </Tabs.TabPane>
        )}
        <Tabs.TabPane tab='YAML' key='2' icon={<FontAwesomeIcon icon={faFileLines} />}>
          <ManifestPreview object={promotion} height='500px' />
        </Tabs.TabPane>
      </Tabs>
    </Modal>
  );
};
