import { useLocalStorage } from '@ui/utils/use-local-storage';

const embeddedArgoCDKey = 'embedded-argocd';

// User preference for how ArgoCD links behave when the embedded ArgoCD
// extension is installed. Enabled (the default) keeps links inside Kargo;
// disabled sends them straight to the ArgoCD instance the Stage is sharded to.
export const useEmbeddedArgoCD = () => useLocalStorage<boolean>(embeddedArgoCDKey, true);
