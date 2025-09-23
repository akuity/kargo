import { useLocalStorage } from '@ui/utils/use-local-storage';

export const useStarProjects = () => {
  const [starred, setStarred] = useLocalStorage('starred-projects', [] as string[]) as [
    string[],
    (ids: string[]) => void
  ];

  const toggleStar = (projectId: string) => {
    if (starred?.includes(projectId)) {
      setStarred(starred.filter((id) => id !== projectId));
      return;
    }
    setStarred([...(starred || []), projectId]);
  };

  return [starred, toggleStar] as const;
};
