import React from 'react';
import clsx from 'clsx';
import styles from './styles.module.css';
function transformImgClassName(className) {
  return clsx(className, styles.img);
}
export default function MDXImg(props) {
  return (
    // eslint-disable-next-line jsx-a11y/alt-text
    <img
      loading="lazy"
      {...props}
      className={transformImgClassName(props.className)}
    />
  );
}
