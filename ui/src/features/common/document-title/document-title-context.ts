import React from 'react';

export type DocumentTitleContextType = {
  /**
   * Formats an array of title parts into the final document title string.
   * Parts are ordered from most-specific to least-specific (e.g. ["Stage: prod", "my-project"]).
   * The default implementation appends "Kargo" and joins with " | ".
   */
  formatTitle: (parts: string[]) => string;
};

const defaultFormatTitle = (parts: string[]) => [...parts, 'Kargo'].join(' | ');

export const DocumentTitleContext = React.createContext<DocumentTitleContextType>({
  formatTitle: defaultFormatTitle
});
