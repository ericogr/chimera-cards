import React from 'react';
import SettingsMenu from './SettingsMenu';

interface UserProfileProps {
  user: {
    name?: string;
    email?: string;
    picture?: string;
  };
  onLogout: () => void;
}

const UserProfile: React.FC<UserProfileProps> = ({ user, onLogout }) => {
  return (
    <div>
      <h2>Authentication Successful!</h2>
      {user.picture && <img src={user.picture} alt="Profile" style={{ borderRadius: '50%' }} />}
      <p><strong>Name:</strong> {user.name}</p>
      <p><strong>Email:</strong> {user.email}</p>
      <div style={{ marginTop: 12 }}>
        <SettingsMenu onLogout={onLogout} />
      </div>
    </div>
  );
};

export default UserProfile;
