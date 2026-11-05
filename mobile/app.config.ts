import type { ConfigContext, ExpoConfig } from 'expo/config';

const DEFAULT_API = 'https://match.heros-agent.space';

export default ({ config }: ConfigContext): ExpoConfig => {
  // Set by hand once an EAS project exists. Without it the app simply doesn't
  // register for push.
  const projectId = process.env.EAS_PROJECT_ID;

  return {
    ...config,
    name: 'SynAgent',
    slug: 'synagent',
    scheme: 'synagent',
    version: '1.3.3',
    orientation: 'portrait',
    icon: './assets/brand/icon.png',
    userInterfaceStyle: 'automatic',
    backgroundColor: '#EDE5FF',
    platforms: ['ios', 'android', 'web'],
    // Served by the API server at /app (internal/httpapi/webapp.go).
    web: { bundler: 'metro', output: 'single', favicon: './assets/brand/favicon.png' },
    experiments: { ...config.experiments, baseUrl: '/app' },
    ios: {
      bundleIdentifier: 'space.herosagent.synagent',
      supportsTablet: false,
      infoPlist: { ITSAppUsesNonExemptEncryption: false },
    },
    android: {
      package: 'space.herosagent.synagent',
      adaptiveIcon: {
        foregroundImage: './assets/brand/adaptive-foreground.png',
        backgroundImage: './assets/brand/adaptive-background.png',
        monochromeImage: './assets/brand/adaptive-monochrome.png',
      },
      versionCode: Number(process.env.ANDROID_VERSION_CODE) || 1,
      softwareKeyboardLayoutMode: 'resize',
      // Lets the app hand a new release to the system installer. Android still asks the person.
      permissions: ['android.permission.REQUEST_INSTALL_PACKAGES'],
      blockedPermissions: ['android.permission.RECORD_AUDIO'],
    },
    plugins: [
      'expo-router',
      'expo-status-bar',
      [
        'expo-splash-screen',
        {
          image: './assets/brand/splash.png',
          imageWidth: 200,
          resizeMode: 'contain',
          backgroundColor: '#EDE5FF',
          dark: { image: './assets/brand/splash.png', backgroundColor: '#1F1145' },
        },
      ],
      'expo-secure-store',
      'expo-image',
      'expo-web-browser',
      'expo-sharing',
      '@react-native-community/datetimepicker',
      'expo-font',
      [
        'expo-image-picker',
        {
          photosPermission: 'SynAgent uses your photos so you can add them to your profile.',
          cameraPermission: false,
          microphonePermission: false,
        },
      ],
      ['expo-notifications', { icon: './assets/brand/notification-icon.png', color: '#7A45F5' }],
      // Playback only: never ask for the microphone, and stop when the app leaves the foreground.
      ['expo-audio', { microphonePermission: false, recordAudioAndroid: false, enableBackgroundPlayback: false }],
      './plugins/withDebugNetworking',
      'expo-iap',
    ],
    extra: {
      apiUrl: process.env.EXPO_PUBLIC_API_URL || DEFAULT_API,
      ...(projectId ? { eas: { projectId } } : {}),
    },
  };
};
