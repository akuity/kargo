import { faClose, faPlus } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Input, Button, Flex, Alert } from 'antd';
import React, { useState, useEffect } from 'react';

type ObjectEditorProps = {
  value?: Record<string, string>;
  onChange?: (value: Record<string, string>) => void;
  keyPlaceholder?: string;
  valuePlaceholder?: string;
};

type InternalRow = {
  id: number;
  key: string;
  val: string;
};

export const ObjectEditor: React.FC<ObjectEditorProps> = ({
  value = {},
  onChange,
  keyPlaceholder = 'Key',
  valuePlaceholder = 'Value'
}) => {
  const [duplicatedKey, setDuplicatedKey] = React.useState(false);
  const [rows, setRows] = useState<InternalRow[]>(() =>
    Object.entries(value).map(([k, v], idx) => ({ id: idx, key: k, val: v }))
  );

  // Sync internal state with external value changes
  useEffect(() => {
    if (Object.keys(value).length === 0 && rows.length > 0 && rows[0].key === '') return;
  }, [value]);

  const triggerChange = (updatedRows: InternalRow[]) => {
    const result: Record<string, string> = {};
    updatedRows.forEach((row) => {
      if (row.key) result[row.key] = row.val;
    });
    onChange?.(result);
  };

  const handleAddField = () => {
    const newRows = [...rows, { id: Date.now(), key: '', val: '' }];
    setRows(newRows);
    triggerChange(newRows);
  };

  const handleRemoveField = (id: number) => {
    const newRows = rows.filter((row) => row.id !== id);
    setRows(newRows);
    triggerChange(newRows);
    detectDuplicateKeys(newRows);
  };

  const handleUpdate = (id: number, field: 'key' | 'val', newValue: string) => {
    const newRows = rows.map((row) => (row.id === id ? { ...row, [field]: newValue } : row));
    setRows(newRows);
    triggerChange(newRows);
  };

  const detectDuplicateKeys = (internalRows: InternalRow[]) => {
    const keys = internalRows.map((row) => row.key).filter((key) => key !== '');
    const uniqueKeys = new Set(keys);
    setDuplicatedKey(uniqueKeys.size !== keys.length);
  };

  return (
    <Flex vertical gap={16}>
      {rows.map((row) => (
        <Flex key={row.id} gap={8} align='center'>
          <Flex flex={1} gap={16}>
            <Input
              placeholder={keyPlaceholder}
              value={row.key}
              onChange={(e) => handleUpdate(row.id, 'key', e.target.value)}
              onBlur={() => detectDuplicateKeys(rows)}
            />
            <Input.TextArea
              placeholder={valuePlaceholder}
              value={row.val}
              onChange={(e) => handleUpdate(row.id, 'val', e.target.value)}
              rows={1}
            />
          </Flex>
          <Button
            type='text'
            icon={<FontAwesomeIcon icon={faClose} />}
            onClick={() => handleRemoveField(row.id)}
            size='small'
            shape='circle'
          />
        </Flex>
      ))}
      <Button type='dashed' onClick={handleAddField} block icon={<FontAwesomeIcon icon={faPlus} />}>
        Add
      </Button>
      {duplicatedKey && <Alert message='The key must be unique' type='error' showIcon banner />}
    </Flex>
  );
};
