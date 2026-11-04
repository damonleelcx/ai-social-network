import { useEffect, useState } from 'react';
import { Pressable, View } from 'react-native';
import { useScreenSize } from '@/ui/screenSize';
import * as api from '@/api/endpoints';
import type { Photo } from '@/api/types';
import { pickAndUploadPhoto } from '@/lib/photos';
import { useLive } from '@/state/live';
import { useSession } from '@/state/session';
import { useTheme } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';
import { Backdrop, Page, ScreenHeader } from '@/ui/layout';
import { MediaImage } from '@/ui/person';
import { Button, ErrorText, Glass, Pill, T } from '@/ui/primitives';
import { confirm, Sheet, SheetAction, showError } from '@/ui/sheet';
import { SukiLoading, SukiSays } from '@/ui/suki';
import { tr } from '@/i18n';

const MAX_PHOTOS = 6;

const STATUS: Record<Photo['status'], { label: string; tone: 'good' | 'warn' | 'bad' | 'plain' }> = {
  pending: { label: "Suki's looking", tone: 'plain' },
  approved: { label: 'Looks good', tone: 'good' },
  rejected: { label: 'Not this one', tone: 'bad' },
  review: { label: 'A person is checking', tone: 'warn' },
};

export default function Photos() {
  const t = useTheme();
  const { width } = useScreenSize();
  const { refreshMe } = useSession();
  const { photos, loaded, refreshPhotos } = useLive();
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<Photo | null>(null);

  useEffect(() => {
    refreshPhotos();
  }, [refreshPhotos]);

  const size = (width - space.xl * 2 - space.md) / 2;

  const add = async () => {
    setUploading(true);
    setError(null);
    try {
      const r = await pickAndUploadPhoto();
      if (r) {
        await refreshPhotos();
        refreshMe().catch(() => {});
      }
    } catch (e: any) {
      setError(e.message);
    } finally {
      setUploading(false);
    }
  };

  const setMain = async (p: Photo) => {
    setSelected(null);
    try {
      await api.photos.setPrimary(p.id);
      await refreshPhotos();
    } catch (e) {
      showError(e);
    }
  };

  const remove = async (p: Photo) => {
    setSelected(null);
    if (!(await confirm(tr('Remove this photo?'), tr('It comes off your profile right away.'), tr('Remove'), true))) return;
    try {
      await api.photos.remove(p.id);
      await refreshPhotos();
      refreshMe().catch(() => {});
    } catch (e) {
      showError(e);
    }
  };

  return (
    <Backdrop>
      <ScreenHeader title={tr('Photos')} />
      <Page refreshing={false} onRefresh={refreshPhotos}>
        {!loaded.photos && !photos.length ? (
          <SukiLoading line={tr('Getting your photos...')} />
        ) : !photos.length ? (
          <SukiSays
            face="surprised"
            size={120}
            title={tr('No photos yet?')}
            body={tr("One clear photo of your face is worth a thousand bios. I'll take a look and tell you honestly how it comes across.")}
          />
        ) : (
          <>
            <T soft>{tr('Tap a photo to make it your main one, or remove it. I check each one before anyone sees it.')}</T>
            <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: space.md }}>
              {photos.map((p) => {
                const s = STATUS[p.status] ?? STATUS.pending;
                return (
                  <Pressable
                    key={p.id}
                    accessibilityRole="button"
                    accessibilityLabel={tr('Photo. {label}.{value}', { label: tr(s.label), value: p.is_primary ? tr(' Your main photo.') : '' })}
                    onPress={() => setSelected(p)}
                    style={{ width: size, gap: space.xs }}
                  >
                    <View>
                      <MediaImage id={p.id} radiusPx={radius.md} style={{ width: size, height: size * 1.25 }} />
                      {p.is_primary ? <Pill label={tr('Main')} tone="accent" style={{ position: 'absolute', top: 8, left: 8, backgroundColor: t.glassStrong }} /> : null}
                    </View>
                    <Pill label={tr(s.label)} tone={s.tone} />
                    {p.reason ? (
                      <T v="caption" soft>
                        {tr(p.reason)}
                      </T>
                    ) : null}
                    {p.tips ? (
                      <T v="caption" faint>
                        {p.tips}
                      </T>
                    ) : null}
                  </Pressable>
                );
              })}
            </View>
          </>
        )}

        <ErrorText message={error} />
        {photos.length < MAX_PHOTOS ? (
          <Button label={photos.length ? tr('Add another photo') : tr('Add a photo')} icon="image" full loading={uploading} onPress={add} />
        ) : (
          <Glass>
            <T v="small" soft center>{tr("That's the full six. Remove one to add another.")}</T>
          </Glass>
        )}
      </Page>

      <Sheet visible={!!selected} onClose={() => setSelected(null)} title={tr('This photo')}>
        {selected?.status === 'approved' && !selected.is_primary ? (
          <SheetAction icon="ribbon-outline" label={tr('Make it my main photo')} onPress={() => setMain(selected)} />
        ) : null}
        {selected && selected.status !== 'approved' ? (
          <T v="small" soft>{tr("It can be your main photo once it's approved.")}</T>
        ) : null}
        {selected ? <SheetAction icon="trash-outline" label={tr('Remove')} danger onPress={() => remove(selected)} /> : null}
      </Sheet>
    </Backdrop>
  );
}
