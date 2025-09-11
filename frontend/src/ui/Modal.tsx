import React from 'react';

export interface ModalProps {
  onClose?: () => void;
  children?: React.ReactNode;
  className?: string;
  contentClassName?: string;
}

const Modal: React.FC<ModalProps> = ({ onClose, children, className, contentClassName }) => {
  return (
    <div className={`modal-overlay ${className || ''}`} onClick={onClose}>
      <div className={`modal-content ${contentClassName || ''}`} onClick={(e) => e.stopPropagation()}>
        {children}
      </div>
    </div>
  );
};

export default Modal;

