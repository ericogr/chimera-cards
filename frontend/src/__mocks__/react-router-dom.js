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

// Minimal hook implementations for tests
module.exports.useLocation = () => ({ pathname: '/' });
module.exports.useNavigate = () => () => {};
module.exports.useParams = () => ({});
module.exports.Link = ({ to, children }) => React.createElement('a', { href: to }, children);
