import { faPlus } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Input, Tag, Space, TagProps } from 'antd';
import { useEffect, useState } from 'react';

export const MultiStringEditor = ({
  value,
  onChange,
  placeholder,
  label,
  className
}: {
  value: string[];
  onChange: (value: string[]) => void;
  placeholder?: string;
  label?: string;
  className?: string;
}) => {
  const [values, _setValues] = useState(value);
  const [newValue, setNewValue] = useState('');

  const setValues = (values: string[]) => {
    _setValues(values);
    onChange(values);
  };

  const addValue = () => {
    if (!newValue || newValue === '') return;
    setValues([...(values || []), newValue]);
    setNewValue('');
  };

  // necessary for form to be reset properly
  useEffect(() => {
    _setValues(value);
  }, [value]);

  const _Tag = (props: TagProps) => (
    <Tag className='py-1 px-2 text-sm' {...props}>
      {props.children}
    </Tag>
  );

  return (
    <div className={className}>
      <div className='flex items-center h-8'>
        {label && <div className='text-xs uppercase font-semibold text-neutral-600'>{label}</div>}
        <div className='ml-auto flex items-center'>
          {values?.length > 1 && (
            <div
              className='text-xs text-neutral-400 cursor-pointer mr-2'
              onClick={() => setValues([])}
            >
              Clear All
            </div>
          )}
        </div>
      </div>
      <div className='rounded bg-neutral-800 p-2'>
        <div className='flex items-center mb-2 min-h-8 flex-wrap gap-2'>
          {(values || []).map((v, i) => (
            <_Tag
              key={i}
              closable
              onClose={() => {
                setValues(values.filter((_, j) => i !== j));
                onChange(values.filter((_, j) => i !== j));
              }}
            >
              <span style={{ paddingRight: '1px' }}>{v}</span>
            </_Tag>
          ))}

          {(values || []).length === 0 && (
            <div className='text-neutral-600 text-sm mx-auto'>Type below to add values</div>
          )}
        </div>

        <div className='flex items-center w-full'>
          <Space.Compact className='w-full'>
            <Input
              value={newValue}
              placeholder={placeholder}
              onChange={(e) => {
                setNewValue(e.target.value);
              }}
              onSubmit={addValue}
              onPressEnter={addValue}
            />
            <Button type='primary' onClick={addValue}>
              <FontAwesomeIcon icon={faPlus} />
            </Button>
          </Space.Compact>
        </div>
      </div>
    </div>
  );
};
