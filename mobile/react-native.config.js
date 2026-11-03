// npm installs these as peers of expo-router, but the app uses none of them (it has
// the native stack and JS tabs, with Animated for motion). Keeping them out of the native
// build saves minutes, and react-native-worklets can't compile from a typical Windows
// checkout at all: its object paths run past the 260-character limit.
const notLinked = { platforms: { android: null, ios: null } };

module.exports = {
  dependencies: {
    'react-native-worklets': notLinked,
    'react-native-reanimated': notLinked,
    'react-native-gesture-handler': notLinked,
  },
};
