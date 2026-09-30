import { zodResolver } from '@hookform/resolvers/zod';
import { Card, Checkbox, Flex, Modal, Switch, Typography } from 'antd';
import { FormProvider, useForm } from 'react-hook-form';

import { useIsAnyExtensionLoaded } from '@ui/extensions/utils';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { SelectorFields } from '@ui/features/common/selector/selector-fields';
import { PromotionPolicy } from '@ui/gen/api/v2/models';

import {
  formValuesFromPromotionPolicy,
  promotionPhases,
  promotionPolicyFromFormValues,
  promotionPolicyFormSchema,
  PromotionPolicyFormValues,
  verificationPhases
} from './promotion-policy-form-utils';

const phaseOptions = (phases: readonly string[]) =>
  phases.map((phase) => ({ label: phase, value: phase }));

type PromotionPolicyModalProps = ModalComponentProps & {
  editing?: boolean;
  promotionPolicy?: PromotionPolicy;
  onSubmit: (promotionPolicy: PromotionPolicy) => void;
};

export const PromotionPolicyModal = ({
  visible,
  hide,
  editing,
  promotionPolicy,
  onSubmit
}: PromotionPolicyModalProps) => {
  // auto-rollback is a Kargo Enterprise field, ignored by OSS -- the same signal
  // that gates the rest of the enterprise-only settings decides whether to show it
  const autoRollbackEditable = useIsAnyExtensionLoaded();

  const form = useForm<PromotionPolicyFormValues>({
    defaultValues: formValuesFromPromotionPolicy(promotionPolicy),
    resolver: zodResolver(promotionPolicyFormSchema)
  });

  const autoRollbackEnabled = form.watch('autoRollbackEnabled');

  const handleSubmit = form.handleSubmit((values) => {
    onSubmit(
      promotionPolicyFromFormValues(values, { autoRollbackEditable, existing: promotionPolicy })
    );
    hide();
  });

  return (
    <FormProvider {...form}>
      <Modal
        open={visible}
        onCancel={hide}
        width={680}
        title={editing ? 'Edit Promotion Policy' : 'New Promotion Policy'}
        okText={editing ? 'Save changes' : 'Create'}
        onOk={handleSubmit}
      >
        <div className='mb-6'>
          <SelectorFields subject='stage' owner='policy' />
        </div>

        <FieldContainer control={form.control} name='autoPromotionEnabled'>
          {({ field }) => (
            <Card size='small'>
              <Flex align='center' justify='space-between' gap={12}>
                <Flex vertical>
                  <Typography.Text strong>Auto-promotion</Typography.Text>
                  <Typography.Text type='secondary' className='text-xs'>
                    Promote new Freight into the matched Stages without waiting for anyone to ask.
                  </Typography.Text>
                </Flex>
                <Switch checked={field.value} onChange={field.onChange} />
              </Flex>
            </Card>
          )}
        </FieldContainer>

        {autoRollbackEditable && (
          <>
            <FieldContainer control={form.control} name='autoRollbackEnabled'>
              {({ field }) => (
                <Card size='small'>
                  <Flex align='center' justify='space-between' gap={12}>
                    <Flex vertical>
                      <Typography.Text strong>Auto-rollback</Typography.Text>
                      <Typography.Text type='secondary' className='text-xs'>
                        Return the matched Stages to their last verified Freight when a promotion or
                        verification ends badly.
                      </Typography.Text>
                    </Flex>
                    <Switch checked={field.value} onChange={field.onChange} />
                  </Flex>
                </Card>
              )}
            </FieldContainer>

            {autoRollbackEnabled && (
              <Flex vertical gap={16} className='mt-4 ml-1'>
                <FieldContainer
                  control={form.control}
                  name='onVerification'
                  label='Roll back on verification'
                  description='Verification outcomes that trigger a rollback.'
                  formItemClassName='!mb-0'
                >
                  {({ field }) => (
                    <Checkbox.Group {...field} options={phaseOptions(verificationPhases)} />
                  )}
                </FieldContainer>

                <FieldContainer
                  control={form.control}
                  name='onPromotion'
                  label='Roll back on promotion'
                  description='Promotion outcomes that trigger a rollback. A failed promotion often means a transient problem with the deployment rather than bad Freight, so leaving these unchecked is the safer default.'
                  formItemClassName='!mb-0'
                >
                  {({ field }) => (
                    <Checkbox.Group {...field} options={phaseOptions(promotionPhases)} />
                  )}
                </FieldContainer>
              </Flex>
            )}
          </>
        )}
      </Modal>
    </FormProvider>
  );
};
