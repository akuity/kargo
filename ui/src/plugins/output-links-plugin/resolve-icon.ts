import { IconDefinition } from '@fortawesome/fontawesome-svg-core';
import * as brandIcons from '@fortawesome/free-brands-svg-icons';
import * as solidIcons from '@fortawesome/free-solid-svg-icons';

export function resolveIcon(name?: string): IconDefinition {
  if (!name) return solidIcons.faLink;
  return (
    (solidIcons[name as keyof typeof solidIcons] as IconDefinition) ??
    (brandIcons[name as keyof typeof brandIcons] as IconDefinition) ??
    solidIcons.faLink
  );
}
