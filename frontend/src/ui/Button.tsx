import React from 'react';

export type ButtonVariant = 'primary' | 'ghost' | 'danger' | 'link';

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
}

const Button: React.FC<ButtonProps> = ({ variant = 'primary', className, children, ...props }) => {
  const variantClass = variant === 'ghost' ? 'btn-ghost' : variant === 'danger' ? 'btn-danger' : variant === 'link' ? 'link-button' : '';
  const classes = [variantClass, className].filter(Boolean).join(' ');
  return (
    <button className={classes || undefined} {...props}>
      {children}
    </button>
  );
};

export default Button;
