import React from 'react';
import Button, { ButtonProps } from './Button';

export interface IconButtonProps extends ButtonProps {
  icon?: string;
  iconAlt?: string;
}

const IconButton: React.FC<IconButtonProps> = ({ icon, iconAlt = '', children, className, ...props }) => {
  return (
    <Button className={`icon-btn ${className || ''}`} {...props}>
      {icon && <img src={icon} alt={iconAlt} className="btn-icon" />}
      {children}
    </Button>
  );
};

export default IconButton;

