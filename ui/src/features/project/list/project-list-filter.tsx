import { faMagnifyingGlass } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Input } from 'antd';
import { ChangeEvent, useEffect, useRef, useState } from 'react';

// How long typing has to pause before the list is searched again.
const SEARCH_DEBOUNCE_MS = 250;

type ProjectListFilterProps = {
  initialValue?: string;
  // the search to show, once typing has paused
  onChange: (filter: string) => void;
  className?: string;
};

// ProjectListFilter searches the project list as the reader types. Clearing
// the search, or pressing Enter, takes effect at once rather than waiting out
// the pause.
export const ProjectListFilter = ({
  initialValue,
  onChange,
  className
}: ProjectListFilterProps) => {
  const [text, setText] = useState(initialValue || '');
  const timer = useRef<number | undefined>(undefined);

  useEffect(() => () => window.clearTimeout(timer.current), []);

  const handleChange = (e: ChangeEvent<HTMLInputElement>) => {
    const next = e.target.value;
    setText(next);
    window.clearTimeout(timer.current);
    if (!next) {
      onChange('');
      return;
    }
    timer.current = window.setTimeout(() => onChange(next), SEARCH_DEBOUNCE_MS);
  };

  const handleSubmit = () => {
    window.clearTimeout(timer.current);
    onChange(text);
  };

  return (
    <Input
      allowClear
      aria-label='Search projects'
      placeholder='Search projects'
      prefix={
        <FontAwesomeIcon
          icon={faMagnifyingGlass}
          className='mr-1 text-xs text-[color:var(--kargo-color-text-tertiary)]'
        />
      }
      className={className}
      value={text}
      onChange={handleChange}
      onPressEnter={handleSubmit}
    />
  );
};
