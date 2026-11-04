// A browser has no keychain, so this is localStorage. The page's Content-Security-Policy
// (internal/httpapi/webapp.go) is what keeps other people's scripts away from the token.
export async function getItemAsync(key: string) {
  return window.localStorage.getItem(key);
}
export async function setItemAsync(key: string, value: string) {
  window.localStorage.setItem(key, value);
}
export async function deleteItemAsync(key: string) {
  window.localStorage.removeItem(key);
}
