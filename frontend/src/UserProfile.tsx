import React from 'react';
import { Button } from './ui';

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
      {user.picture && <img src={user.picture} alt="Profile" className="avatar-circle" />}
      <p><strong>Name:</strong> {user.name}</p>
      <p><strong>Email:</strong> {user.email}</p>
      <div className="mt-12">
        <Button onClick={onLogout}>Logout</Button>
      </div>
    </div>
  );
};

export default UserProfile;
