import React from 'react';

export interface TextareaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {}

const Textarea: React.FC<TextareaProps> = ({ className, ...props }) => {
  return <textarea className={`form-textarea ${className || ''}`} {...props} />;
};

export default Textarea;

