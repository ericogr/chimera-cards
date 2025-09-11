import React from 'react';

export interface RowProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: 'between' | 'center' | 'center-sm' | 'start';
}

const Row: React.FC<RowProps> = ({ variant = 'center', className, children, ...props }) => {
  const map: Record<string, string> = {
    between: 'row-between',
    center: 'row-center',
    'center-sm': 'row-center-sm',
    start: 'row-start',
  };
  const classes = [map[variant], className].filter(Boolean).join(' ');
  return (
    <div className={classes || undefined} {...props}>
      {children}
    </div>
  );
};

export default Row;

