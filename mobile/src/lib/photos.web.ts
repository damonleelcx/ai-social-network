import { ImageManipulator, SaveFormat } from 'expo-image-manipulator';
import * as ImagePicker from 'expo-image-picker';
import { API_URL, ApiError, authHeaders, getToken, handleFailure, OFFLINE_MESSAGE } from '@/api/client';

const MAX_EDGE = 1600;

/** photos.ts, except the upload: a browser sends the file with fetch. */
export async function pickAndUploadPhoto(): Promise<{ id: string; status: string } | null> {
  const picked = await ImagePicker.launchImageLibraryAsync({ mediaTypes: ['images'], quality: 1, exif: false });
  if (picked.canceled || !picked.assets?.[0]) return null;
  const asset = picked.assets[0];

  let ctx = ImageManipulator.manipulate(asset.uri);
  if (asset.width > MAX_EDGE || asset.height > MAX_EDGE) {
    ctx = asset.width >= asset.height ? ctx.resize({ width: MAX_EDGE }) : ctx.resize({ height: MAX_EDGE });
  }
  const rendered = await ctx.renderAsync();
  const saved = await rendered.saveAsync({ format: SaveFormat.JPEG, compress: 0.86 });

  const sentWith = getToken();
  let res: Response;
  try {
    const body = await (await fetch(saved.uri)).blob();
    res = await fetch(`${API_URL}/v1/photos`, {
      method: 'POST',
      headers: { ...authHeaders(), 'Content-Type': 'image/jpeg' },
      body,
    });
  } catch {
    throw new ApiError(0, OFFLINE_MESSAGE);
  }
  const data = await res.json().catch(() => undefined);
  if (!res.ok) throw handleFailure(res.status, typeof data?.error === 'string' ? data.error : '', sentWith);
  return data;
}

/** photos.ts's selfie, from the browser's camera where it offers one. */
export async function takeVerificationSelfie(): Promise<{ verified: boolean; reason?: string } | null> {
  const shot = await ImagePicker.launchCameraAsync({ mediaTypes: ['images'], cameraType: ImagePicker.CameraType.front, quality: 1, exif: false });
  if (shot.canceled || !shot.assets?.[0]) return null;
  const rendered = await ImageManipulator.manipulate(shot.assets[0].uri).resize({ width: 1080 }).renderAsync();
  const saved = await rendered.saveAsync({ format: SaveFormat.JPEG, compress: 0.86 });
  const sentWith = getToken();
  let res: Response;
  try {
    const body = await (await fetch(saved.uri)).blob();
    res = await fetch(`${API_URL}/v1/verify/selfie`, { method: 'POST', headers: { ...authHeaders(), 'Content-Type': 'image/jpeg' }, body });
  } catch {
    throw new ApiError(0, OFFLINE_MESSAGE);
  }
  const data = await res.json().catch(() => undefined);
  if (!res.ok) throw handleFailure(res.status, typeof data?.error === 'string' ? data.error : '', sentWith);
  return data;
}
