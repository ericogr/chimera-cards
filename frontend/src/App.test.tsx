import React from 'react';
import { render, screen } from '@testing-library/react';
import App from './App';

test('renders login prompt when not authenticated', () => {
  render(<App />);
  const prompt = screen.getByText(/please log in to continue/i);
  expect(prompt).toBeInTheDocument();
  const btn = screen.getByRole('button', { name: /sign in with google/i });
  expect(btn).toBeInTheDocument();
});
