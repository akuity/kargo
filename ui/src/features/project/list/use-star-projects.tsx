import { useLocalStorage } from '@ui/utils/use-local-storage';

export const useStarProjects = () => {
  const [starred, setStarred] = useLocalStorage<string[]>('starred-projects', []);

  const toggleStar = (projectId: string) => {
    if (starred?.includes(projectId)) {
      setStarred(starred.filter((id) => id !== projectId));
      return;
    }
    setStarred([...(starred || []), projectId]);
  };

  return [starred, toggleStar] as const;
};
