// Google Play Billing doesn't exist in a browser, and the web build stays free of the library.
export const isPlayBuild = false;

export type PlayPurchase = { productId: string; token: string; pending: boolean };

export async function playPrices(_ids: string[]): Promise<Record<string, string>> {
  return {};
}

export async function buyOnPlay(_productId: string, _accountTag: string): Promise<PlayPurchase | null> {
  return null;
}

export async function unfinishedPlayPurchases(): Promise<PlayPurchase[]> {
  return [];
}
