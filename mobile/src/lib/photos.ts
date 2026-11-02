import { File } from 'expo-file-system';
import { ImageManipulator, SaveFormat } from 'expo-image-manipulator';
import * as ImagePicker from 'expo-image-picker';
import { API_URL, ApiError, authHeaders, getToken, handleFailure, OFFLINE_MESSAGE } from '@/api/client';

const MAX_EDGE = 1600;

/**
 * Lets the person pick a photo, shrinks it to at most 1600px, converts it to JPEG
 * (which is how HEIC gets through) and uploads the raw bytes. Returns null if they
 * backed out of the picker.
 */
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
  let result: Awaited<ReturnType<File['upload']>>;
  try {
    result = await new File(saved.uri).upload(`${API_URL}/v1/photos`, {
      httpMethod: 'POST',
      mimeType: 'image/jpeg',
      headers: { ...authHeaders(), 'Content-Type': 'image/jpeg' },
    });
  } catch {
    throw new ApiError(0, OFFLINE_MESSAGE);
  }
  let data: any;
  try {
    data = JSON.parse(result.body ?? '');
  } catch {
    data = undefined;
  }
  if (result.status < 200 || result.status >= 300) {
    throw handleFailure(result.status, typeof data?.error === 'string' ? data.error : '', sentWith);
  }
  return data;
}
