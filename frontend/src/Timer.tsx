import React from 'react';

interface Props {
  // Remaining time in seconds. If null/undefined the component renders nothing.
  seconds?: number | null;
  // Threshold in seconds below which the timer should blink. Default 60s.
  threshold?: number;
  // Additional CSS class to apply to the wrapper.
  className?: string;
  style?: React.CSSProperties;
}

const Timer: React.FC<Props> = ({ seconds, threshold = 60, className, style }) => {
  if (seconds == null) return null;
  const sec = Math.max(0, Math.floor(seconds));
  const minutes = Math.floor(sec / 60);
  const s = sec % 60;
  const formatted = `${minutes}:${String(s).padStart(2, '0')}`;
  const shouldBlink = sec > 0 && sec < threshold;
  const classes = `${shouldBlink ? 'blink-text' : ''}${className ? ' ' + className : ''}`.trim();
  return (
    <span className={classes || undefined} style={style}>
      {formatted}
    </span>
  );
};

export default Timer;

