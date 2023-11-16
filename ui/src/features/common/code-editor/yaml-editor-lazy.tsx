import Editor, { loader } from '@monaco-editor/react';
import { Typography } from 'antd';
import type { JSONSchema4 } from 'json-schema';
import * as monaco from 'monaco-editor';
import { configureMonacoYaml } from 'monaco-yaml';
import React, { FC, useEffect, useRef } from 'react';

import styles from './yaml-editor.module.less';

loader.config({ monaco });

export interface YamlEditorProps {
  value: string;
  disabled?: boolean;
  onChange?(value: string | undefined): void;
  className?: string;
  width?: string;
  height?: string;
  schema?: JSONSchema4;
  placeholder?: string;
}

const YamlEditor: FC<YamlEditorProps> = (props) => {
  const editorRef = useRef<monaco.editor.IStandaloneCodeEditor | null>(null);
  const { value, disabled, onChange, className, width, height, schema, placeholder } = props;

  const handleOnChange = (value: string | undefined) => {
    onChange?.(value);
  };

  useEffect(() => {
    configureMonacoYaml(monaco, {
      enableSchemaRequest: true,
      hover: true,
      completion: true,
      validate: true,
      isKubernetes: true,
      format: true,
      schemas: schema && [
        {
          uri: 'http://myserver/foo-schema.json',
          fileMatch: ['*'],
          schema
        }
      ]
    });
  }, []);

  const handleEditorDidMount = (editor: monaco.editor.IStandaloneCodeEditor) => {
    editorRef.current = editor;
  };

  return (
    <>
      <div
        style={{ border: '1px solid #d9d9d9', height, overflow: 'hidden' }}
        className={className}
      >
        <Editor
          options={{
            readOnly: disabled,
            lineDecorationsWidth: 5,
            lineNumbersMinChars: 0,
            glyphMargin: false,
            folding: false,
            lineNumbers: 'off',
            minimap: {
              enabled: false
            },
            fontSize: 11
          }}
          width={width}
          height={height}
          language='yaml'
          value={value}
          onChange={handleOnChange}
          onMount={handleEditorDidMount}
        />

        {placeholder && (
          <p
            className={`${styles.placeholderWrapper} font-mono`}
            onClick={() => {
              editorRef.current?.focus?.();
            }}
          >
            {!value &&
              placeholder
                ?.trim()
                ?.split('\n')
                .map((line, i) => (
                  <React.Fragment key={i}>
                    {line
                      .split('')
                      .map((char, j) =>
                        char === ' ' ? <React.Fragment key={j}>&nbsp;</React.Fragment> : char
                      )}
                    <br />
                  </React.Fragment>
                ))}
          </p>
        )}
      </div>
      {!props.disabled && (
        <div className='mt-1'>
          <Typography.Text type='secondary'>
            Press <strong>ctrl + space</strong> to show suggestions
          </Typography.Text>
        </div>
      )}
    </>
  );
};

export default YamlEditor;
