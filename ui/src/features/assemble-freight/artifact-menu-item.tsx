import classNames from 'classnames';

import { DiscoveryResult } from './types';
import { getSubscriptionKey, isEqualSubscriptions } from './unique-subscription-key';

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
    {items.map((item) => {
      const isSelected = !!selected && isEqualSubscriptions(selected, item);

      return (
        <ArtifactMenuItem
          key={getSubscriptionKey(item)}
          onClick={() => onClick(item)}
          selected={isSelected}
        >
          {item.repoURL}
          {item.$typeName === 'github.com.akuity.kargo.api.v1alpha1.ChartDiscoveryResult' &&
            `/${item.name}`}
        </ArtifactMenuItem>
      );
    })}
  </>
);
