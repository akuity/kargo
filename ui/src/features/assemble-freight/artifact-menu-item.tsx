import classNames from 'classnames';

import { DiscoveryResult } from './types';

export interface ArtifactMenuItemProps {
  onClick: () => void;
  selected: boolean;
  children: React.ReactNode;
}

export const ArtifactMenuItem = ({ onClick, selected, children }: ArtifactMenuItemProps) => (
  <div
    onClick={onClick}
    className={classNames(
      'p-2 bg-white mb-1 cursor-pointer rounded-md border border-solid border-gray-100 break-words',
      { 'border-sky-500': selected }
    )}
  >
    {children}
  </div>
);

export const ArtifactMenuItems = <T extends DiscoveryResult>({
  onClick,
  selected,
  items,
  getKey
}: {
  onClick: (item: T) => void;
  selected?: T;
  items: T[];
  getKey: (item: T) => string;
}) => (
  <>
    {items.map((item) => {
      const key = getKey(item);
      const isSelected = !!selected && getKey(selected) === key;

      return (
        <ArtifactMenuItem key={key} onClick={() => onClick(item)} selected={isSelected}>
          {key}
        </ArtifactMenuItem>
      );
    })}
  </>
);
