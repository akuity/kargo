import { useContext, useEffect } from 'react';

import { DocumentTitleContext } from './document-title-context';

/**
 * Sets the browser tab title (document.title) from an array of parts.
 * Parts should be ordered from most-specific to least-specific,
 * e.g. ["Stage: prod", "my-project"] → "Stage: prod | my-project | Kargo"
 *
 * Falsy parts are filtered out, so callers can safely pass undefined/null
 * params without extra guards.
 */
export const useDocumentTitle = (parts: (string | undefined | null | false)[]) => {
  const { appName } = useContext(DocumentTitleContext);

  useEffect(() => {
    const filtered = parts.filter(Boolean) as string[];
    document.title = [...filtered, appName].join(' - ');
  }, [parts, appName]);
};
