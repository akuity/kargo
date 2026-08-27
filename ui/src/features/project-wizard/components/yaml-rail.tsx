import { faCheck, faCopy } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { theme } from 'antd';
import { useEffect, useRef, useState } from 'react';

import { YamlEditor } from '@ui/features/common/code-editor/yaml-editor';

import { ResourceRef, resourceKey } from '../manifest/manifest-builder';

type YamlRailProps = {
  yaml: string;
  stepTitle: string;
  resources: ResourceRef[];
  // When set, the rail is editable and edits sync live into wizard state.
  // Returns an error message to display, or null when the text was applied.
  onLiveEdit?: (text: string) => string | null;
};

const liveEditDebounceMs = 400;
// Monaco drops its tokens on every new document and re-tokenizes asynchronously,
// so a sync per keystroke flashes uncolored text. Settle on a pause instead.
// TODO: remove once Monaco is replaced.
const incomingSyncDebounceMs = 400;

export const YamlRail = ({ yaml, stepTitle, resources, onLiveEdit }: YamlRailProps) => {
  const { token } = theme.useToken();
  const [copied, setCopied] = useState(false);
  const [text, setText] = useState(yaml);
  const [error, setError] = useState<string | null>(null);

  // While the cursor is in the editor the user owns the text: incoming
  // regenerated YAML (echoes of their own edits) must not reset it.
  const focusedRef = useRef(false);
  const debounceRef = useRef<number | undefined>(undefined);
  const pendingRef = useRef(false);
  const onLiveEditRef = useRef(onLiveEdit);
  onLiveEditRef.current = onLiveEdit;
  const textRef = useRef(text);
  textRef.current = text;

  useEffect(() => {
    if (focusedRef.current) {
      return;
    }
    const id = window.setTimeout(() => {
      setText(yaml);
      setError(null);
    }, incomingSyncDebounceMs);
    return () => window.clearTimeout(id);
  }, [yaml]);

  useEffect(() => () => window.clearTimeout(debounceRef.current), []);

  const commit = () => {
    pendingRef.current = false;
    setError(onLiveEditRef.current?.(textRef.current) ?? null);
  };

  const handleChange = (value: string | undefined) => {
    setText(value ?? '');
    pendingRef.current = true;
    window.clearTimeout(debounceRef.current);
    debounceRef.current = window.setTimeout(commit, liveEditDebounceMs);
  };

  const handleBlur = () => {
    focusedRef.current = false;
    // Flush an in-flight edit instead of leaving it behind, but don't
    // commit when the user just clicked in and out without typing.
    window.clearTimeout(debounceRef.current);
    if (pendingRef.current) {
      commit();
    }
  };

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 1300);
    } catch (_) {
      // clipboard unavailable (e.g. insecure context) -- silently ignore
    }
  };

  return (
    <aside className='w-[380px] shrink-0 flex flex-col bg-[#0f1722] text-[#d6deea]'>
      <div className='flex items-start justify-between px-5 pt-4 pb-3 border-b border-white/10'>
        <div>
          <div className='text-[10px] font-semibold tracking-widest text-[#8fa1b8]'>
            LIVE YAML {onLiveEdit ? 'EDITOR' : 'PREVIEW'}
          </div>
          <div className='text-xs mt-0.5'>
            {stepTitle}
            {onLiveEdit && <span className='text-[#8fa1b8]'> — edits sync with the form</span>}
          </div>
        </div>
        <button
          className='flex items-center gap-1.5 rounded border border-white/20 bg-transparent px-2 py-1 text-xs text-[#d6deea] cursor-pointer hover:bg-white/10'
          onClick={copy}
        >
          <FontAwesomeIcon icon={copied ? faCheck : faCopy} size='xs' />
          {copied ? 'Copied' : 'Copy'}
        </button>
      </div>
      <div className='flex-1 min-h-0'>
        <YamlEditor
          value={text}
          disabled={!onLiveEdit}
          onChange={onLiveEdit ? handleChange : undefined}
          onFocus={() => {
            focusedRef.current = true;
          }}
          onBlur={handleBlur}
          theme='dark'
          height='100%'
        />
      </div>
      {error && (
        <div
          className='border-t border-white/10 px-5 py-2 text-xs'
          style={{ color: token.colorError }}
        >
          Not applied: {error}
        </div>
      )}
      <div className='border-t border-white/10 px-5 py-3'>
        <div className='text-xs text-[#8fa1b8] mb-2'>
          {resources.length} resource{resources.length === 1 ? '' : 's'} will be applied
        </div>
        <div className='flex flex-wrap gap-1.5'>
          {resources.map((r) => (
            <span
              key={resourceKey(r)}
              className='inline-flex items-center gap-1.5 rounded-full bg-white/10 px-2.5 py-0.5 font-mono text-[11px]'
            >
              <span
                className='w-1.5 h-1.5 rounded-full'
                style={{ backgroundColor: token.colorSuccess }}
              />
              {r.kind}/{r.name}
            </span>
          ))}
        </div>
      </div>
    </aside>
  );
};
