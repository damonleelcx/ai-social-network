import { Platform } from 'react-native';

const WEB_CLIENT_ID = process.env.EXPO_PUBLIC_GOOGLE_WEB_CLIENT_ID ?? '';

/** Android only, and only in a build given a client id. */
export const googleSignInAvailable = Platform.OS === 'android' && !!WEB_CLIENT_ID;

/** A fresh token from Google, or null if they backed out. */
export async function googleIdToken(): Promise<string | null> {
  // Loaded here so that it's only ever touched on Android.
  const { GoogleSignin, isCancelledResponse, isSuccessResponse } = await import('@react-native-google-signin/google-signin');
  GoogleSignin.configure({ webClientId: WEB_CLIENT_ID });
  await GoogleSignin.hasPlayServices();
  // Always show the account picker, so the right account can be chosen on a shared phone.
  await GoogleSignin.signOut().catch(() => {});
  const res = await GoogleSignin.signIn();
  if (isCancelledResponse(res) || !isSuccessResponse(res)) return null;
  return res.data.idToken;
}
