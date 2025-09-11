import React from 'react';

export interface PageProps extends React.HTMLAttributes<HTMLElement> {
  compact?: boolean;
}

const Page: React.FC<PageProps> = ({ compact = false, className, children, ...props }) => {
  return (
    <main className={`page-main ${compact ? 'page-main--compact' : ''} ${className || ''}`} {...props}>
      {children}
    </main>
  );
};

export default Page;

