import React from 'react';
import ReactDOM from 'react-dom/client';
import { GoogleOAuthProvider } from '@react-oauth/google';
import './index.css';
import './responsive.css';
import App from './App';
import reportWebVitals from './reportWebVitals';

import { BrowserRouter } from 'react-router-dom';

const root = ReactDOM.createRoot(
  document.getElementById('root') as HTMLElement
);

const runtimeClientId = ((window as any)._env_ && (window as any)._env_.REACT_APP_GOOGLE_CLIENT_ID) || process.env.REACT_APP_GOOGLE_CLIENT_ID;

function renderApp() {
  if (runtimeClientId) {
    return (
      <React.StrictMode>
        <BrowserRouter>
          <GoogleOAuthProvider clientId={runtimeClientId}>
            <App />
          </GoogleOAuthProvider>
        </BrowserRouter>
      </React.StrictMode>
    );
  }

  // If clientId is missing the Google SDK would throw. Render the app
  // without the provider so the UI loads; login flows will remain
  // disabled until a valid client id is provided at runtime.
  // eslint-disable-next-line no-console
  console.warn('REACT_APP_GOOGLE_CLIENT_ID is not set; Google login disabled.');
  return (
    <React.StrictMode>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </React.StrictMode>
  );
}

root.render(renderApp());

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();
