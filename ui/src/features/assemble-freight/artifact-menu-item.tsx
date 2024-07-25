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

export const ArtifactMenuItems = ({
  onClick,
  selected,
  items
}: {
  onClick: (item: DiscoveryResult) => void;
  selected?: DiscoveryResult;
  items: DiscoveryResult[];
}) => (
  <>
    {items.map((item) => (
      <ArtifactMenuItem
        key={item.repoURL}
        onClick={() => onClick(item)}
        selected={selected?.repoURL === item.repoURL}
      >
        {item.repoURL}
      </ArtifactMenuItem>
    ))}
  </>
);
