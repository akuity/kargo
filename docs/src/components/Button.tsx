import clsx from 'clsx';
import style from './Button.module.scss';
import { ButtonHTMLAttributes, PropsWithChildren } from 'react';

type ButtonProps = {
    btnType?: 'default' | 'primary'
};

export const Button = (props: PropsWithChildren<ButtonProps & ButtonHTMLAttributes<{}>>) => {
    let {btnType, className, ...nativeProps} = props;
    
    btnType = btnType || 'default';

    className = clsx(className, style.btn, btnType === 'primary' && style.primary);

    return <button {...props} className={className} />
};