export const ConfirmPromotionDialogue = ({
  stageName,
  promotionType,
  onClick
}: {
  stageName: string;
  promotionType: string;
  onClick: () => void;
}) => {
  const style = 'rounded text-white font-semibold p-1 cursor-pointer';
  return (
    <div>
      <div className='text-xs px-2 mb-3'>
        Are you sure you want to promote{' '}
        {promotionType === 'subscribers' ? ' subscribers' : <b>{stageName}</b>}?
      </div>
      <div className='flex items-center w-full justify-center'>
        <div className={`${style} bg-sky-500 mr-4 hover:bg-sky-600`} onClick={onClick}>
          YES
        </div>
        <div className={`${style} bg-sky-800 hover:bg-sky-900`}>NO</div>
      </div>
    </div>
  );
};
