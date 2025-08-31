const React = require('react');

function Routes({ children }) {
  return React.createElement('div', null, children);
}

function Route({ element }) {
  return element || null;
}

function BrowserRouter({ children }) {
  return React.createElement('div', null, children);
}

module.exports = {
  Routes,
  Route,
  BrowserRouter,
};

