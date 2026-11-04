// Google sign-in is for the Android app, and the web build stays free of its library.
export const googleSignInAvailable = false;

export async function googleIdToken(): Promise<string | null> {
  return null;
}
