import { getBackground } from '@ui/utils/stages';

export const Palette = () => (
  <div className='flex items-center p-6 bg-zinc-900'>
    {Array.from('x'.repeat(15)).map((_, i) => (
      <div key={i} className={`h-6 w-6 mr-4 rounded-md ${getBackground(i)}`} />
    ))}
  </div>
);
