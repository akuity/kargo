import { useParams } from 'react-router-dom';

export const Environment = () => {
  const { name } = useParams();
  return <div>{name}</div>;
};
