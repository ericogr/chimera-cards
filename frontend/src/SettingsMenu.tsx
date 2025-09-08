import React, { useEffect, useRef, useState } from 'react';
import './SettingsMenu.css';

interface SettingsMenuProps {
  onLogout: () => void;
  onProfile?: () => void;
  className?: string;
}

const SettingsMenu: React.FC<SettingsMenuProps> = ({ onLogout, onProfile, className }) => {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const onDocClick = (e: MouseEvent) => {
      if (!ref.current) return;
      if (e.target instanceof Node && !ref.current.contains(e.target)) {
        setOpen(false);
      }
    };
    const onEsc = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false);
    };
    document.addEventListener('click', onDocClick);
    document.addEventListener('keydown', onEsc);
    return () => {
      document.removeEventListener('click', onDocClick);
      document.removeEventListener('keydown', onEsc);
    };
  }, []);

  return (
    <div ref={ref} className={`settings-menu ${className || ''}`}>
      <button
        aria-label="Open settings"
        className="settings-toggle"
        onClick={(e) => { e.stopPropagation(); setOpen(o => !o); }}
      >
        <svg width="20" height="16" viewBox="0 0 20 16" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden>
          <rect x="0" y="1" width="20" height="2" rx="1" fill="currentColor" />
          <rect x="0" y="7" width="20" height="2" rx="1" fill="currentColor" />
          <rect x="0" y="13" width="20" height="2" rx="1" fill="currentColor" />
        </svg>
      </button>
      {open && (
        <div className="settings-dropdown" role="menu">
          {onProfile && (
            <button
              className="settings-item"
              role="menuitem"
              onClick={() => { setOpen(false); onProfile(); }}
            >
              User Profile
            </button>
          )}
          <button
            className="settings-item"
            role="menuitem"
            onClick={() => {
              setOpen(false);
              onLogout();
            }}
          >
            Logout
          </button>
        </div>
      )}
    </div>
  );
};

export default SettingsMenu;
