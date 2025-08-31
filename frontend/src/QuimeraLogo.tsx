import React from 'react';

const QuimeraLogo: React.FC = () => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    viewBox="0 0 200 200"
    aria-labelledby="quimera-logo-title"
    role="img"
    className="quimera-logo"
  >
    <title id="quimera-logo-title">Quimera Cards Logo</title>
    <defs>
      <linearGradient id="lionGradient" x1="0%" y1="0%" x2="100%" y2="100%">
        <stop offset="0%" style={{ stopColor: '#ffc107', stopOpacity: 1 }} />
        <stop offset="100%" style={{ stopColor: '#e91e63', stopOpacity: 1 }} />
      </linearGradient>
      <linearGradient id="goatGradient" x1="0%" y1="0%" x2="100%" y2="100%">
        <stop offset="0%" style={{ stopColor: '#9e9e9e', stopOpacity: 1 }} />
        <stop offset="100%" style={{ stopColor: '#607d8b', stopOpacity: 1 }} />
      </linearGradient>
      <linearGradient id="snakeGradient" x1="0%" y1="0%" x2="100%" y2="100%">
        <stop offset="0%" style={{ stopColor: '#4caf50', stopOpacity: 1 }} />
        <stop offset="100%" style={{ stopColor: '#8bc34a', stopOpacity: 1 }} />
      </linearGradient>
    </defs>

    {/* Lion Head */}
    <path
      d="M 50,50 Q 70,30 100,50 T 150,50 L 150,100 Q 100,120 50,100 Z"
      fill="url(#lionGradient)"
    />
    <circle cx="80" cy="70" r="5" fill="white" />
    <circle cx="120" cy="70" r="5" fill="white" />

    {/* Goat Horns */}
    <path
      d="M 70,40 Q 60,20 50,10 C 40,20 50,40 70,40 Z"
      fill="url(#goatGradient)"
    />
    <path
      d="M 130,40 Q 140,20 150,10 C 160,20 150,40 130,40 Z"
      fill="url(#goatGradient)"
    />

    {/* Snake Tail */}
    <path
      d="M 100,120 Q 80,140 100,160 T 120,180"
      stroke="url(#snakeGradient)"
      strokeWidth="10"
      strokeLinecap="round"
      fill="none"
    />
  </svg>
);

export default QuimeraLogo;
