import React from 'react';
import { Button, Avatar } from './ui';

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
      {user.picture && <Avatar src={user.picture} alt={user.name || 'Profile'} size={80} />}
      <p><strong>Name:</strong> {user.name}</p>
      <p><strong>Email:</strong> {user.email}</p>
      <div className="mt-12">
        <Button onClick={onLogout}>Logout</Button>
      </div>
    </div>
  );
};

export default UserProfile;
