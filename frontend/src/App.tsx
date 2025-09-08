import React, { useState, useEffect } from 'react';
import { Routes, Route } from 'react-router-dom';
import { useGoogleLogin, CodeResponse } from '@react-oauth/google';
import { apiFetch } from './api';
import Lobby from './Lobby';
import GameRoom from './GameRoom';
import GameBoard from './GameBoard';
import ProfilePage from './ProfilePage';
import Header from './Header';
import './App.css';
import * as constants from './constants';
import { useLocation } from 'react-router-dom';

interface User {
  name?: string;
  email?: string;
  picture?: string;
}

const App: React.FC = () => {
  const [user, setUser] = useState<User | null>(null);
  const [error, setError] = useState<string | null>(null);
  const location = useLocation();

  // Runtime configuration provided at container start (generated from REACT_APP_* environment variables)
  const runtimeConfig = (window as any)._env_ || {};
  const requiredRuntimeKeys = ['REACT_APP_GOOGLE_CLIENT_ID', 'REACT_APP_API_BASE_URL'];
  const missingRuntimeKeys = requiredRuntimeKeys.filter(k => !(runtimeConfig[k] && String(runtimeConfig[k]).trim() !== ''));

  useEffect(() => {
    const storedUser = localStorage.getItem('user');
    if (storedUser) {
      try {
        setUser(JSON.parse(storedUser));
      } catch (e) {
        console.error("Failed to parse user from localStorage", e);
        localStorage.removeItem('user');
      }
    }
  }, []);

  const handleLoginSuccess = async (codeResponse: Omit<CodeResponse, 'error' | 'error_description' | 'error_uri'>) => {
    try {
      const response = await apiFetch(constants.API_AUTH_GOOGLE_CALLBACK, {
        method: 'POST',
        headers: { [constants.HEADER_CONTENT_TYPE]: constants.CONTENT_TYPE_JSON },
        body: JSON.stringify({ code: codeResponse.code }),
      });
      if (response.ok) {
        const userInfo = await response.json();
        setUser(userInfo);
        localStorage.setItem('user', JSON.stringify(userInfo));
        try { localStorage.setItem('session_ok', '1'); } catch {}
        setError(null);
      } else {
        throw new Error('Backend code exchange failed');
      }
    } catch (err) {
      setError('Failed to log in. Please try again.');
      setUser(null);
      localStorage.removeItem('user');
    }
  };

  const login = useGoogleLogin({
    flow: 'auth-code',
    onSuccess: handleLoginSuccess,
    onError: () => setError('Google authentication failed. Please try again.'),
  });

  const handleLogout = () => {
    setUser(null);
    localStorage.removeItem('user');
    try { localStorage.removeItem('session_ok'); } catch {}
    setError(null);
  };

  if (!user) {
    return (
      <div className="App">
        <header className="App-header">
          {missingRuntimeKeys.length > 0 && (
            <div className="runtime-config-warning">
              Runtime configuration incomplete: missing {missingRuntimeKeys.join(', ')}. Set the corresponding
              `REACT_APP_` environment variables for the frontend container (for example, in `docker-compose.yml`).
            </div>
          )}
          <img src="/welcome_logo.png" alt="Welcome Logo" className="welcome-logo" />
          <div>
            <p>Please log in to continue</p>
            {error && <p className="error-message">{error}</p>}
            <button onClick={() => login()} className="google-login-button">
              Sign in with Google
            </button>
          </div>
        </header>
      </div>
    );
  }

  return (
    <div className="main-app-container">
      {missingRuntimeKeys.length > 0 && (
        <div className="runtime-config-warning">
          Runtime configuration incomplete: missing {missingRuntimeKeys.join(', ')}. Set the corresponding
          `REACT_APP_` environment variables for the frontend container (for example, in `docker-compose.yml`).
        </div>
      )}
      <Header user={user} onLogout={handleLogout} showProfileOption={location.pathname === '/'} />
      <Routes>
        <Route path="/" element={<Lobby user={user} onLogout={handleLogout} />} />
        <Route path="/profile" element={<ProfilePage user={user} onLogout={handleLogout} />} />
        <Route path="/game/:gameId" element={<GameRoom />} />
        <Route path="/game/:gameId/board" element={<GameBoard />} />
      </Routes>
    </div>
  );
};

export default App;
