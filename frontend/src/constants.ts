// Centralized frontend constants for API endpoints and headers
export const API_PREFIX = '/api';

export const API_ENTITIES = `${API_PREFIX}/entities`;
export const API_ENTITIES_IMAGE = `${API_PREFIX}/entities/image`;
export const API_ASSETS_ENTITIES = `${API_PREFIX}/assets/entities`;
export const API_ASSETS_HYBRIDS = `${API_PREFIX}/assets/hybrids`;
export const API_PUBLIC_GAMES = `${API_PREFIX}/public-games`;
export const API_LEADERBOARD = `${API_PREFIX}/leaderboard`;
export const API_GAMES = `${API_PREFIX}/games`;
export const API_GAMES_JOIN = `${API_GAMES}/join`;
export const API_PLAYER_STATS = `${API_PREFIX}/player-stats`;
// The Google OAuth callback is mounted at the root `/auth/...` path so
// external OAuth redirects can reach it directly (not under `/api`).
export const API_AUTH_GOOGLE_CALLBACK = `/auth/google/oauth2callback`;

// Headers and content types
export const HEADER_CONTENT_TYPE = 'Content-Type';
export const CONTENT_TYPE_JSON = 'application/json';

export const API_CONFIG = `${API_PREFIX}/config`;
