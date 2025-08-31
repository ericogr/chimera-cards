// Minimal mock for @react-oauth/google used in tests
function useGoogleLogin(opts) {
  return function() {
    if (opts && typeof opts.onError === 'function') {
      // do nothing
    }
  };
}

const CodeResponse = {};

module.exports = {
  useGoogleLogin,
  CodeResponse,
};

