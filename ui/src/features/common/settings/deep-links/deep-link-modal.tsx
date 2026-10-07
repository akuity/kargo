import { zodResolver } from '@hookform/resolvers/zod';
import { Input, Modal } from 'antd';
import { useState } from 'react';
import { useForm } from 'react-hook-form';

import { FieldContainer } from '@ui/features/common/form/field-container';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { DeepLink } from '@ui/gen/api/v2/models';

import {
  deepLinkFormSchema,
  deepLinkFromFormValues,
  DeepLinkFormValues,
  formValuesFromDeepLink
} from './deep-link-form-utils';
import { DeepLinkKind, deepLinkKinds } from './deep-link-kinds';

// AntD caps a tooltip at 250px, on the root rather than the body. That is
// narrower than the examples, which wrapped mid-token.
const expressionHelpProps = { styles: { root: { maxWidth: 380 } } };

// plain elements rather than Typography: this renders inside AntD's dark
// tooltip, where Typography's own color would land dark on dark
const ExpressionHelp = ({ syntax, examples }: { syntax: string; examples: string[] }) => (
  <div className='text-xs'>
    {syntax}
    {examples.map((example) => (
      <div key={example} className='mt-1 font-mono'>
        {example}
      </div>
    ))}
  </div>
);

type DeepLinkModalProps = ModalComponentProps & {
  kind: DeepLinkKind;
  editing?: boolean;
  link?: DeepLink;
  onSubmit: (link: DeepLink) => Promise<unknown>;
};

export const DeepLinkModal = ({
  visible,
  hide,
  kind,
  editing,
  link,
  onSubmit
}: DeepLinkModalProps) => {
  const { resource, templateRoot, expressionRoot, urlExamples, conditionExamples } =
    deepLinkKinds[kind];

  const form = useForm<DeepLinkFormValues>({
    defaultValues: formValuesFromDeepLink(link),
    resolver: zodResolver(deepLinkFormSchema)
  });

  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = form.handleSubmit(async (values) => {
    setSubmitting(true);
    try {
      await onSubmit(deepLinkFromFormValues(values));
      hide();
    } catch {
      // already reported by the mutation layer; just keep the form open with
      // the values still in it
    } finally {
      setSubmitting(false);
    }
  });

  return (
    <Modal
      open={visible}
      onCancel={submitting ? undefined : hide}
      width={680}
      title={editing ? `Edit ${resource} Link` : `New ${resource} Link`}
      okText={editing ? 'Save changes' : 'Create'}
      onOk={handleSubmit}
      okButtonProps={{ loading: submitting }}
    >
      <FieldContainer
        control={form.control}
        name='title'
        label='Title'
        required
        description='The label shown in the Links menu.'
      >
        {({ field }) => <Input {...field} placeholder='Runbook' />}
      </FieldContainer>

      <FieldContainer
        control={form.control}
        name='url'
        label='URL'
        required
        description={`A Go template evaluated against ${templateRoot}. Sprig functions are available.`}
        tooltip={
          <ExpressionHelp
            syntax={`Anything on the ${resource} is reachable from ${templateRoot}. Fields that are not set are absent rather than empty, so reach optional ones with get. Indexing a missing field drops the whole link.`}
            examples={urlExamples}
          />
        }
        tooltipProps={expressionHelpProps}
      >
        {({ field }) => (
          <Input.TextArea
            {...field}
            autoSize={{ minRows: 1, maxRows: 4 }}
            placeholder={`https://example.com/${kind}/{{ ${templateRoot}.metadata.name }}`}
          />
        )}
      </FieldContainer>

      <FieldContainer
        control={form.control}
        name='description'
        label='Description'
        description='Optional. Shown as a tooltip on the link.'
      >
        {({ field }) => <Input {...field} placeholder='Operational runbook for this service' />}
      </FieldContainer>

      <FieldContainer
        control={form.control}
        name='if'
        label='Condition'
        description={`Optional. An expression over ${expressionRoot}. The link is hidden unless it evaluates to true.`}
        tooltip={
          <ExpressionHelp
            syntax={`An expression returning true or false, with no {{ }} around it. The ${resource} is bound to ${expressionRoot}. Reach optional fields with ?. Indexing or len() on a missing one drops the link.`}
            examples={conditionExamples}
          />
        }
        tooltipProps={expressionHelpProps}
        formItemClassName='!mb-0'
      >
        {({ field }) => (
          <Input.TextArea
            {...field}
            autoSize={{ minRows: 1, maxRows: 4 }}
            placeholder={conditionExamples[0]}
          />
        )}
      </FieldContainer>
    </Modal>
  );
};
