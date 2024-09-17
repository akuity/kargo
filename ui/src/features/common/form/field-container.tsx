import { faInfoCircle } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex, Form, FormItemProps, Tooltip } from 'antd';
import React from 'react';
import {
  FieldValues,
  useController,
  UseControllerProps,
  UseControllerReturn
} from 'react-hook-form';

interface Props<T extends FieldValues> extends UseControllerProps<T> {
  children: (props: UseControllerReturn<T>) => React.ReactNode;
  label?: string;
  formItemOptions?: Omit<FormItemProps, 'label'>;
  className?: string;
  formItemClassName?: string;
  description?: string;
  tooltip?: React.ReactNode;
  required?: boolean;
}

export const FieldContainer = <T extends FieldValues>({
  children,
  label,
  formItemOptions,
  className,
  formItemClassName,
  description,
  tooltip,
  required,
  ...props
}: Props<T>) => {
  const controller = useController(props);

  return (
    <Form layout='vertical' component='div' className={className}>
      <Form.Item
        {...{
          ...formItemOptions,
          label: label && (
            <Flex align='center'>
              {label}
              {required && (
                <Tooltip title='Required' placement='right'>
                  <span className='text-red-500 ml-1'>*</span>
                </Tooltip>
              )}
              {tooltip && (
                <Tooltip title={tooltip} placement='top'>
                  <FontAwesomeIcon icon={faInfoCircle} className='ml-2' />
                </Tooltip>
              )}
            </Flex>
          )
        }}
        className={formItemClassName}
        help={controller.fieldState.error?.message}
        validateStatus={controller.fieldState.error?.message ? 'error' : ''}
      >
        {description && <div className='text-xs text-gray-500 mb-2 -mt-1'>{description}</div>}
        {children(controller)}
      </Form.Item>
    </Form>
  );
};
