import React from 'react';

export interface FormInputProps extends React.InputHTMLAttributes<HTMLInputElement> {}

const FormInput: React.FC<FormInputProps> = ({ className, ...props }) => {
  return <input className={`form-input ${className || ''}`} {...props} />;
};

export default FormInput;

