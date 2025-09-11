import React from 'react';

export interface AvatarProps extends React.ImgHTMLAttributes<HTMLImageElement> {
  size?: number | string;
}

const Avatar: React.FC<AvatarProps> = ({ size = 40, className, style, ...props }) => {
  const px = typeof size === 'number' ? `${size}px` : size;
  const altText = (props.alt as string) ?? '';
  const rest = { ...props } as React.ImgHTMLAttributes<HTMLImageElement>;
  return (
    <img
      {...rest}
      alt={altText}
      className={`avatar-circle ${className || ''}`}
      style={{ height: px, width: px, objectFit: 'cover', ...style }}
    />
  );
};

export default Avatar;
