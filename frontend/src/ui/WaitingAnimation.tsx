import React from 'react';
import './WaitingAnimation.css';

interface Props {
  size?: number;
  alt?: string;
  src?: string;
}

const WaitingAnimation: React.FC<Props> = ({ size = 160, alt = 'Waiting for chimera', src = '/waiting_for_chimera.apng' }) => {
  const style: React.CSSProperties = { width: size, height: size };
  return (
    <div className="waiting-animation-wrapper">
      <div className="waiting-animation-mask" style={style}>
        <img src={src} alt={alt} className="waiting-animation-media" />
      </div>
    </div>
  );
};

export default WaitingAnimation;

