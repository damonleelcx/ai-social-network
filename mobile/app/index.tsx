import { Redirect } from 'expo-router';
import { useSession } from '@/state/session';

/** The root route only decides where to go, from GET /v1/me. */
export default function Index() {
  const { status } = useSession();
  if (status === 'signedOut') return <Redirect href="/welcome" />;
  if (status === 'unverified') return <Redirect href="/verify" />;
  if (status === 'ready') return <Redirect href="/suki" />;
  return null;
}
