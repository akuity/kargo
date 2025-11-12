import { useExtensionsContext } from './extensions-context';

// Some extensions might take some time to load (e.g. dynamic extensions)
// This hook checks if at least one extension is loaded, so we know whether to show the loader or redirect elsewhere
export const useIsAnyExtensionLoaded = () => {
  const extensions = useExtensionsContext();

  return Object.values(extensions).some((ext) => {
    if (Array.isArray(ext)) {
      return ext.length > 0;
    }

    return !!ext;
  });
};
