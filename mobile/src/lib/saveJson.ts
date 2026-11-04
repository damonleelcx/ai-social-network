import { File, Paths } from 'expo-file-system';
import * as Sharing from 'expo-sharing';
import { Share } from '@/lib/dialogs';

/** Hands a JSON file to the person: the share sheet, or the text itself if there isn't one. */
export async function saveJson(name: string, json: string, dialogTitle: string) {
  const file = new File(Paths.cache, name);
  if (file.exists) file.delete();
  file.create();
  file.write(json);
  if (await Sharing.isAvailableAsync()) {
    await Sharing.shareAsync(file.uri, { mimeType: 'application/json', dialogTitle });
  } else {
    await Share.share({ message: json });
  }
}
