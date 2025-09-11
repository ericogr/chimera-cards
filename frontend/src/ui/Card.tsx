import React from 'react';

export interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  children?: React.ReactNode;
}

const Card: React.FC<CardProps> = ({ children, className, ...props }) => {
  return (
    <div className={`card ${className || ''}`} {...props}>
      {children}
    </div>
  );
};

export default Card;

