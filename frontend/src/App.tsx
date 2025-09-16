import React, { useState, useEffect } from 'react';
import { Routes, Route, useLocation } from 'react-router-dom';
import { useGoogleLogin, CodeResponse } from '@react-oauth/google';
import { apiFetch, apiJson } from './api';
import Lobby from './Lobby';
import GameRoom from './GameRoom';
import GameBoard from './GameBoard';
import ProfilePage from './ProfilePage';
import Header from './Header';
import { Button } from './ui';
import './App.css';
import * as constants from './constants';
import { getMissingRuntimeKeys, safeSetLocal, safeRemoveLocal } from './runtimeConfig';
// Use window.location when Router is not available (tests)

interface User {
  name?: string;
  email?: string;
  picture?: string;
}

const App: React.FC = () => {
  const [user, setUser] = useState<User | null>(null);
  const [error, setError] = useState<string | null>(null);
  const location = useLocation();
  const requiredRuntimeKeys = ['REACT_APP_GOOGLE_CLIENT_ID', 'REACT_APP_API_BASE_URL'];
  const missingRuntimeKeys = getMissingRuntimeKeys(requiredRuntimeKeys);
  const [versionInfo, setVersionInfo] = useState<{version?: string; commit?: string; date?: string; dirty?: string} | null>(null);

  useEffect(() => {
    const storedUser = localStorage.getItem('user');
    if (storedUser) {
      try {
        setUser(JSON.parse(storedUser));
      } catch (e) {
        console.error('Failed to parse user from localStorage', e);
        safeRemoveLocal('user');
      }
    }
  }, []);

  useEffect(() => {
    let mounted = true;
    if (user) return; // fetch version only on the login screen
    async function fetchVersion() {
      try {
        const data = await apiJson<{version?: string; commit?: string; date?: string; dirty?: string}>(constants.API_VERSION);
        if (!mounted) return;
        if (data && data.version) setVersionInfo(data);
      } catch (err) {
        // ignore; version is non-critical for login
      }
    }
    fetchVersion();
    return () => { mounted = false; };
  }, [user]);

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
        safeSetLocal('user', JSON.stringify(userInfo));
        safeSetLocal('session_ok', '1');
        setError(null);
      } else {
        throw new Error('Backend code exchange failed');
      }
    } catch (err) {
      setError('Failed to log in. Please try again.');
      setUser(null);
      safeRemoveLocal('user');
    }
  };

  const login = useGoogleLogin({
    flow: 'auth-code',
    onSuccess: handleLoginSuccess,
    onError: () => setError('Google authentication failed. Please try again.'),
  });

  const handleLogout = () => {
    setUser(null);
    safeRemoveLocal('user');
    safeRemoveLocal('session_ok');
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
            <Button className="google-login-button" onClick={() => login()}>
              Sign in with Google
            </Button>
            {versionInfo && (
              <div className="version-discrete">Version: {versionInfo.version}{versionInfo.dirty === 'true' ? '-dirty' : ''} {versionInfo.commit ? `(${versionInfo.commit})` : null}</div>
            )}
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
      <Header user={user} onLogout={handleLogout} showProfileOption={(location && location.pathname) === '/'} />
      <Routes>
        <Route path="/" element={<Lobby user={user} onLogout={handleLogout} />} />
        <Route path="/profile" element={<ProfilePage user={user} onLogout={handleLogout} onUserUpdate={(u) => setUser(u)} />} />
        <Route path="/game/:gameCode" element={<GameRoom />} />
        <Route path="/game/:gameCode/board" element={<GameBoard />} />
      </Routes>
    </div>
  );
};

export default App;
