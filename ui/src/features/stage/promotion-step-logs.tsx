import {
  faCheck,
  faCompress,
  faCopy,
  faExpand,
  faTextWidth
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Editor } from '@monaco-editor/react';
import { Button, Flex, Tooltip, Typography } from 'antd';
import { useMemo, useState } from 'react';

import { useTheme } from '@ui/features/common/theme/use-theme';
import { useLocalStorage } from '@ui/utils/use-local-storage';

import {
  monacoEditorLogLanguage,
  monacoEditorLogLanguageTheme,
  monacoEditorLogLanguageThemeDark,
  useMonacoEditorLogLanguage
} from '../common/analysis-run-logs/use-monaco-editor-log-language';

// Pinned rather than left to Monaco's default so the panel can be sized in
// whole lines.
const fontSizePx = 12;
const lineHeightPx = 18;

// Room for the editor's top and bottom padding.
const chromePx = 16;

// Roughly what the 200px YAML view used to show.
const collapsedLines = 10;

// The point of expanding: a screenful of log at once.
const expandedLines = 50;

// One preference for every log panel rather than per step: the intent is "I
// want to see more log", not a note about one particular step.
const expandedKey = 'promotion-step-logs-expanded';

type StepLogsProps = {
  lines: string[];
};

// A log panel for a step's console output. Uses the same Monaco log language
// and themes as AnalysisRun logs, so both read the same way, and adds an
// expand toggle for reading more than a preview at a time.
export const StepLogs = ({ lines }: StepLogsProps) => {
  const [expanded, setExpanded] = useLocalStorage(expandedKey, false);
  const [wrap, setWrap] = useState(false);
  const [copied, setCopied] = useState(false);

  const { isDark } = useTheme();

  useMonacoEditorLogLanguage();

  const value = useMemo(() => lines.join('\n'), [lines]);

  const expandable = lines.length > collapsedLines;

  // Never taller than the log itself, so a three-line log gets a three-line box.
  const visibleLines = Math.min(lines.length, expanded ? expandedLines : collapsedLines);
  const height = `${visibleLines * lineHeightPx + chromePx}px`;

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (_) {
      // clipboard unavailable (e.g. insecure context) -- silently ignore
    }
  };

  return (
    <div>
      <Flex align='center' gap={4} className='mb-1'>
        <Typography.Text type='secondary' className='text-xs'>
          {lines.length} {lines.length === 1 ? 'line' : 'lines'}
        </Typography.Text>
        <Tooltip title={wrap ? 'Disable line wrapping' : 'Wrap long lines'}>
          <Button
            className='ml-auto'
            size='small'
            type={wrap ? 'default' : 'text'}
            icon={<FontAwesomeIcon icon={faTextWidth} className='text-xs' />}
            onClick={() => setWrap(!wrap)}
          />
        </Tooltip>
        <Tooltip title={copied ? 'Copied' : 'Copy logs'}>
          <Button
            size='small'
            type='text'
            icon={<FontAwesomeIcon icon={copied ? faCheck : faCopy} className='text-xs' />}
            onClick={copy}
          />
        </Tooltip>
        {expandable && (
          <Button
            size='small'
            type='text'
            icon={<FontAwesomeIcon icon={expanded ? faCompress : faExpand} className='text-xs' />}
            onClick={() => setExpanded(!expanded)}
          >
            {expanded ? 'Collapse' : 'Expand'}
          </Button>
        )}
      </Flex>

      <Editor
        language={monacoEditorLogLanguage}
        theme={isDark ? monacoEditorLogLanguageThemeDark : monacoEditorLogLanguageTheme}
        value={value}
        height={height}
        options={{
          readOnly: true,
          // A log is not an editable document: no caret, no current-line
          // highlight, no overview ruler competing with the content.
          domReadOnly: true,
          renderLineHighlight: 'none',
          overviewRulerLanes: 0,
          hideCursorInOverviewRuler: true,
          lineNumbers: 'on',
          wordWrap: wrap ? 'on' : 'off',
          fontSize: fontSizePx,
          lineHeight: lineHeightPx,
          padding: { top: 8, bottom: 8 },
          minimap: { enabled: false },
          folding: false,
          guides: { indentation: false },
          // The height is sized to the content, so trailing blank space would
          // only ever be dead space.
          scrollBeyondLastLine: false
        }}
      />
    </div>
  );
};
