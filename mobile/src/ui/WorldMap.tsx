import { Image } from 'expo-image';
import { useEffect, useState } from 'react';
import { View } from 'react-native';
import * as api from '@/api/endpoints';
import type { UserMap } from '@/api/types';
import { useTheme } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';
import { T } from './primitives';
import { tr } from '@/i18n';

// The picture runs from 84°N to 58°S, drawn with even degrees (see assets/brand/world-dots.png).
const TOP = 84;
const BOTTOM = -58;
export const MAP_ASPECT = 960 / 376;

/** Where people are, in crowds only. If the numbers don't arrive it's still a map. */
export function WorldMap({ width }: { width: number }) {
  const t = useTheme();
  const [data, setData] = useState<UserMap | null>(null);
  const height = width / MAP_ASPECT;

  useEffect(() => {
    let live = true;
    api.publicInfo.userMap().then((d) => live && setData(d)).catch(() => {});
    return () => {
      live = false;
    };
  }, []);

  const cells = data?.cells ?? [];
  const biggest = Math.max(1, ...cells.map((c) => c.n));
  return (
    <View style={{ alignItems: 'center' }} accessible accessibilityRole="image" accessibilityLabel={tr('World map of where people use SynAgent')}>
      <View style={{ width, height }}>
        <Image
          source={require('../../assets/brand/world-dots.png')}
          style={{ width, height, opacity: 0.38 }}
          tintColor={t.ink}
          contentFit="contain"
        />
        {cells.map((c) => {
          const size = (width / 320) * (7 + 9 * Math.sqrt(c.n / biggest));
          return (
            <View
              key={`${c.lat},${c.lon}`}
              style={{
                position: 'absolute',
                left: ((c.lon + 180) / 360) * width - size / 2,
                top: ((TOP - c.lat) / (TOP - BOTTOM)) * height - size / 2,
                width: size,
                height: size,
                borderRadius: size / 2,
                backgroundColor: t.accent,
                borderWidth: 1,
                borderColor: t.glassBorder,
              }}
            />
          );
        })}
      </View>
      {cells.length > 0 && data ? (
        <T v="small" soft center numberOfLines={1} style={{ marginTop: space.xs }}>
          {tr('{count} people have joined', { count: data.total.toLocaleString() })}
        </T>
      ) : null}
    </View>
  );
}
