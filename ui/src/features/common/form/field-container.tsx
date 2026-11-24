import { faInfoCircle } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex, Form, FormItemProps, Tooltip, TooltipProps } from 'antd';
import React from 'react';
import {
  FieldPath,
  FieldValues,
  useController,
  UseControllerProps,
  UseControllerReturn
} from 'react-hook-form';

interface Props<
  T extends FieldValues,
  TName extends FieldPath<T> = FieldPath<T>
> extends UseControllerProps<T, TName> {
  children: (props: UseControllerReturn<T, TName>) => React.ReactNode;
  label?: string;
  formItemOptions?: Omit<FormItemProps, 'label'>;
  className?: string;
  formItemClassName?: string;
  description?: string;
  tooltip?: React.ReactNode;
  tooltipProps?: Omit<TooltipProps, 'title'>;
  required?: boolean;
}

export const FieldContainer = <T extends FieldValues, TName extends FieldPath<T> = FieldPath<T>>({
  children,
  label,
  formItemOptions,
  className,
  formItemClassName,
  description,
  tooltip,
  tooltipProps,
  required,
  ...props
}: Props<T, TName>) => {
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
                <Tooltip title='Required' placement='top'>
                  <span className='text-red-500 ml-1'>*</span>
                </Tooltip>
              )}
              {tooltip && (
                <Tooltip title={tooltip} placement='top' {...tooltipProps}>
                  <FontAwesomeIcon
                    icon={faInfoCircle}
                    className='ml-1 text-xs text-gray-500 dark:text-neutral-400'
                  />
                </Tooltip>
              )}
            </Flex>
          )
        }}
        className={formItemClassName}
        help={controller.fieldState.error?.message}
        validateStatus={controller.fieldState.error?.message ? 'error' : ''}
      >
        {description && (
          <div className='text-xs text-gray-500 dark:text-neutral-400 mb-2 -mt-1'>
            {description}
          </div>
        )}
        {children(controller)}
      </Form.Item>
    </Form>
  );
};
