import { useState } from 'react';
import { Pressable, View } from 'react-native';
import type { ReportReason } from '@/api/types';
import { REPORT_REASONS } from '@/lib/format';
import { useTheme } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';
import { Button, ErrorText, Field, Pill, T } from '@/ui/primitives';
import { Sheet } from '@/ui/sheet';
import { tr } from '@/i18n';

export function ReportSheet({
  visible,
  name,
  matched = true,
  onClose,
  onSend,
}: {
  visible: boolean;
  name: string;
  matched?: boolean;
  onClose: () => void;
  onSend: (reason: ReportReason, details: string) => Promise<void>;
}) {
  const t = useTheme();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  return (
    <Sheet visible={visible} onClose={onClose} title={tr('Report {name}', { name: name })}>
      <T soft>{tr("They won't know you reported them. What happened?")}</T>
      <View style={{ gap: 2 }}>
        {REPORT_REASONS.map((r) => (
          <Pressable
            key={r.value}
            accessibilityRole="radio"
            accessibilityState={{ selected: reason === r.value }}
            onPress={() => setReason(r.value)}
            style={{ paddingVertical: space.sm, flexDirection: 'row', alignItems: 'center', gap: space.md }}
          >
            <View
              style={{
                width: 20,
                height: 20,
                borderRadius: 10,
                borderWidth: reason === r.value ? 6 : 1.5,
                borderColor: reason === r.value ? t.accent : t.inkFaint,
              }}
            />
            <T>{tr(r.label)}</T>
          </Pressable>
        ))}
      </View>
      <Field label={tr('Anything else?')} value={details} onChangeText={setDetails} multiline maxLength={2000} hint={tr('Optional, and it helps.')} />
      <ErrorText message={error} />
      <Button
        label={tr('Send report')}
        kind="danger"
        full
        disabled={!reason}
        loading={busy}
        onPress={async () => {
          if (!reason) return;
          setBusy(true);
          setError(null);
          try {
            await onSend(reason, details.trim());
            setReason(null);
            setDetails('');
          } catch (e: any) {
            setError(e.message);
          } finally {
            setBusy(false);
          }
        }}
      />
      {/* Kept visible so it's clear reporting and leaving are separate choices. */}
      {matched ? <Pill tone="plain" label={tr("Reporting doesn't end the match. Block or unmatch for that.")} /> : null}
    </Sheet>
  );
}
