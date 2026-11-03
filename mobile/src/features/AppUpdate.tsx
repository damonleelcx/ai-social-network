import * as FileSystem from 'expo-file-system/legacy';
import * as IntentLauncher from 'expo-intent-launcher';
import { useEffect, useState } from 'react';
import { Linking, View } from 'react-native';
import { DOWNLOAD_PAGE, findUpdate, type AppRelease } from '@/lib/appUpdate';
import { space } from '@/theme/tokens';
import { Button, T } from '@/ui/primitives';
import { Sheet, showError } from '@/ui/sheet';
import { tr } from '@/i18n';

const FLAG_GRANT_READ_URI_PERMISSION = 1;

/**
 * Offers a newer release once per launch. Android downloads it and hands it to the system
 * installer, which always asks the person to confirm. An iPhone can't replace a sideloaded
 * app from inside it, so there the button opens the download page.
 */
export function AppUpdate() {
  const [release, setRelease] = useState<AppRelease | null>(null);
  const [percent, setPercent] = useState<number | null>(null);

  useEffect(() => {
    findUpdate().then(setRelease);
  }, []);

  if (!release) return null;
  const { apkUrl, version } = release;

  const install = async () => {
    if (!apkUrl) {
      Linking.openURL(DOWNLOAD_PAGE).catch(showError);
      return;
    }
    setPercent(0);
    try {
      const download = FileSystem.createDownloadResumable(apkUrl, `${FileSystem.cacheDirectory}SynAgent-${version}.apk`, {}, (p) => {
        if (p.totalBytesExpectedToWrite > 0) setPercent(Math.round((p.totalBytesWritten / p.totalBytesExpectedToWrite) * 100));
      });
      const file = await download.downloadAsync();
      if (!file || file.status !== 200) throw new Error(tr("The download didn't finish. Please try again."));
      const data = await FileSystem.getContentUriAsync(file.uri);
      await IntentLauncher.startActivityAsync('android.intent.action.VIEW', {
        data,
        type: 'application/vnd.android.package-archive',
        flags: FLAG_GRANT_READ_URI_PERMISSION,
      });
    } catch (e) {
      showError(e);
    } finally {
      setPercent(null);
    }
  };

  return (
    <Sheet visible onClose={() => setRelease(null)} title={tr('Version {version} is ready', { version })}>
      <T soft>
        {apkUrl
          ? tr('One tap downloads it. Android then asks you to confirm the install, and your account stays as it is.')
          : tr("An iPhone can't update this app from inside it. Download the new file and sign it with your sideloading tool again.")}
      </T>
      <View style={{ gap: space.sm }}>
        <Button
          label={percent === null ? (apkUrl ? tr('Update now') : tr('Get the new version')) : tr('Downloading {percent}%', { percent })}
          icon="download-outline"
          full
          disabled={percent !== null}
          onPress={install}
        />
        <Button label={tr('Later')} kind="ghost" full disabled={percent !== null} onPress={() => setRelease(null)} />
      </View>
    </Sheet>
  );
}
