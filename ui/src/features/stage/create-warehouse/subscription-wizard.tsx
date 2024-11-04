import { PlainMessage } from '@bufbuild/protobuf';
import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { faEye, faTrash, IconDefinition } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import Form from '@rjsf/antd';
import validator from '@rjsf/validator-ajv8';
import { Button, Card, Collapse, Modal, Select, Tag, Typography } from 'antd';
import AntdFormLabel from 'antd/es/form/FormItemLabel';
import classNames from 'classnames';
import { JSONSchema7 } from 'json-schema';
import { useState } from 'react';

import { DescriptionFieldTemplate } from '@ui/features/common/form/rjsf/description-field-template';
import { FieldTemplate } from '@ui/features/common/form/rjsf/field-template';
import { ObjectFieldTemplate } from '@ui/features/common/form/rjsf/object-field-template';
import { IconSetByKargoTerminology } from '@ui/features/common/icons';
import { ObjectDescription } from '@ui/features/common/object-description';
import { RepoSubscription } from '@ui/gen/v1alpha1/generated_pb';

import { warehouseCreateFormJSONSchema } from './schema';

const subscriptionTypes = Object.keys(
  // @ts-expect-error err
  warehouseCreateFormJSONSchema.properties.subscriptions.items.properties
);

export const SubscriptionWizard = (props: {
  subscriptions: PlainMessage<RepoSubscription>[];
  onChange(subscriptions: PlainMessage<RepoSubscription>[]): void;
}) => {
  const [selectedNewSubscription, setSelectedNewSubscription] = useState(
    subscriptionTypes[2] /* image as default and common subscription */
  );

  return (
    <>
      <AntdFormLabel prefixCls='' label='Subscriptions' />

      <div className='flex gap-4 my-4 relative'>
        <div
          className={classNames('w-5/12 rounded-md flex gap-y-4 flex-wrap h-fit sticky top-0', {
            'bg-gray-100 text-center p-5 justify-center': props.subscriptions?.length === 0
          })}
        >
          {props.subscriptions.map((subscription, idx) => (
            <SubscriptionWizard.Subscription
              key={idx}
              subscription={subscription}
              onDelete={() =>
                props.onChange(props.subscriptions.filter((_, currentIdx) => currentIdx !== idx))
              }
            />
          ))}
          {!props.subscriptions?.length && (
            <Typography.Text type='secondary'>No subscriptions.</Typography.Text>
          )}
        </div>
        <div className='w-7/12 space-y-4'>
          <Select
            className='w-full'
            options={subscriptionTypes.map((type) => ({ label: type, value: type }))}
            value={selectedNewSubscription}
            onChange={(newSubscription) => setSelectedNewSubscription(newSubscription)}
          />
          <Collapse
            items={[
              {
                label: `Click Here to configure and add ${selectedNewSubscription} subscription`,
                children: (
                  <Form
                    validator={validator}
                    schema={
                      // @ts-expect-error schema.ts override
                      warehouseCreateFormJSONSchema.properties.subscriptions.items.properties[
                        selectedNewSubscription
                      ] as JSONSchema7
                    }
                    templates={{
                      DescriptionFieldTemplate,
                      ObjectFieldTemplate,
                      FieldTemplate,
                      ArrayFieldDescriptionTemplate: () => null
                    }}
                    uiSchema={{
                      'ui:submitButtonOptions': {
                        norender: true
                      }
                    }}
                    onSubmit={(data) =>
                      props.onChange([
                        ...props.subscriptions,
                        {
                          [selectedNewSubscription]: data.formData
                        }
                      ])
                    }
                  >
                    <Button htmlType='submit' icon={<IconSetByKargoTerminology.subscription />}>
                      Add Subscription
                    </Button>
                  </Form>
                )
              }
            ]}
          />
        </div>
      </div>
    </>
  );
};

SubscriptionWizard.Subscription = (props: {
  subscription: PlainMessage<RepoSubscription>;
  className?: string;
  onDelete(): void;
}) => {
  let icon: IconDefinition | null = null;

  // one of git, image or chart
  const subscriptionType = (Object.keys(props.subscription)[0] || '') as
    | keyof PlainMessage<RepoSubscription>
    | '';

  if (subscriptionType === '') {
    return <Tag color='red'>Corrupt Subscription! Please Check YAML</Tag>;
  }

  // @ts-expect-error this field is must thus there won't be any subscription entry without this field
  const subscriptionSource: string = props.subscription[subscriptionType]?.repoURL;

  switch (subscriptionType) {
    case 'git':
      icon = faGit;
      break;
    case 'image':
      icon = faDocker;
      break;
  }

  return (
    <Card
      className={classNames(props.className, 'w-full h-fit')}
      actions={[
        <FontAwesomeIcon
          key='trash'
          icon={faTrash}
          className='text-red-400'
          onClick={props.onDelete}
        />,
        <FontAwesomeIcon
          key='info'
          icon={faEye}
          onClick={() =>
            Modal.confirm({
              width: '756px',
              title: subscriptionType,
              content: <ObjectDescription data={props.subscription[subscriptionType] || {}} />,
              cancelButtonProps: {
                hidden: true
              }
            })
          }
        />
      ]}
    >
      <Card.Meta
        avatar={icon && <FontAwesomeIcon icon={icon} />}
        title={subscriptionType}
        description={subscriptionSource}
      />
    </Card>
  );
};
