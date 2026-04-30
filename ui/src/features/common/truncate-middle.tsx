export const TruncateMiddle = ({ children }: { children: string }) => {
  const start = children.slice(0, children.length - 3);
  const suffix = children.slice(-3).trim();
  return (
    <div className='min-w-0 max-w-[76px] flex flex-shrink'>
      <div className='truncate'>{start}</div>
      <div className='flex-shrink-0'>{suffix}</div>
    </div>
  );
};
