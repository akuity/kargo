import { ColorMapHex, getBackgroundKey } from '@ui/features/stage/utils';

export const Palette = () => (
  <div className='p-6 bg-zinc-900'>
    {Array.from('x'.repeat(15)).map((_, i) => {
      const key = getBackgroundKey(i);
      return (
        <div className='flex items-center mb-2' key={i}>
          <div
            className={`h-6 w-6 mr-4 rounded-md`}
            style={{ backgroundColor: ColorMapHex[key] }}
          />
          <div className='text-sm text-white font-mono ml-2'>{key}</div>
        </div>
      );
    })}
  </div>
);
