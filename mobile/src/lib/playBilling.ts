import {
  ErrorCode,
  fetchProducts,
  getAvailablePurchases,
  initConnection,
  purchaseErrorListener,
  purchaseUpdatedListener,
  requestPurchase,
} from 'expo-iap';
import { Platform } from 'react-native';

/**
 * Only the build that goes to Google Play sells through it. The direct-download APK can't
 * use Play Billing, and the Play build may not sell digital goods any other way.
 */
export const isPlayBuild = Platform.OS === 'android' && process.env.EXPO_PUBLIC_PLAY_STORE === '1';

export type PlayPurchase = { productId: string; token: string; pending: boolean };

let connected: Promise<unknown> | null = null;
function connect() {
  connected ??= initConnection().catch((e) => {
    connected = null;
    throw e;
  });
  return connected;
}

function shape(p: { productId: string; purchaseToken?: string | null; purchaseState?: string }): PlayPurchase | null {
  return p.purchaseToken ? { productId: p.productId, token: p.purchaseToken, pending: p.purchaseState === 'pending' } : null;
}

/** Each product's price, as Play shows it in this person's country. */
export async function playPrices(ids: string[]): Promise<Record<string, string>> {
  await connect();
  const products = await fetchProducts({ skus: ids, type: 'in-app' });
  const out: Record<string, string> = {};
  for (const p of (products ?? []) as { id: string; displayPrice: string }[]) out[p.id] = p.displayPrice;
  return out;
}

/** Null if they backed out. The purchase still has to be checked by the server. */
export async function buyOnPlay(productId: string, accountTag: string): Promise<PlayPurchase | null> {
  await connect();
  return new Promise((resolve, reject) => {
    const done = (fn: () => void) => {
      updated.remove();
      failed.remove();
      fn();
    };
    const updated = purchaseUpdatedListener((p) => {
      if (p.productId === productId) done(() => resolve(shape(p)));
    });
    const failed = purchaseErrorListener((e) => {
      done(() => (e.code === ErrorCode.UserCancelled ? resolve(null) : reject(new Error(e.message))));
    });
    requestPurchase({ request: { google: { skus: [productId], obfuscatedAccountId: accountTag } }, type: 'in-app' }).catch((e) =>
      done(() => reject(e)),
    );
  });
}

/** What's been paid for but never reached the server, because the app closed or lost its connection. */
export async function unfinishedPlayPurchases(): Promise<PlayPurchase[]> {
  await connect();
  const list = await getAvailablePurchases();
  return (list ?? []).map(shape).filter((p): p is PlayPurchase => !!p);
}
