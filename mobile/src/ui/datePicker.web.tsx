import { createElement } from 'react';
import { useTheme } from '@/theme/ThemeProvider';

type Props = {
  value: Date;
  mode?: 'date' | 'time' | 'datetime';
  minimumDate?: Date;
  maximumDate?: Date;
  display?: string;
  onChange: (event: unknown, date?: Date) => void;
};

const pad = (n: number) => String(n).padStart(2, '0');
const day = (d: Date) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
const minute = (d: Date) => `${day(d)}T${pad(d.getHours())}:${pad(d.getMinutes())}`;

/** The browser's own date input, standing in for the iOS picker inside the same sheet. */
export function DateTimePicker({ value, mode = 'date', minimumDate, maximumDate, onChange }: Props) {
  const t = useTheme();
  const format = mode === 'datetime' ? minute : day;
  return createElement('input', {
    type: mode === 'datetime' ? 'datetime-local' : 'date',
    value: format(value),
    min: minimumDate ? format(minimumDate) : undefined,
    max: maximumDate ? format(maximumDate) : undefined,
    style: { font: 'inherit', fontSize: 16, padding: 12, borderRadius: 12, border: `1px solid ${t.accent}`, background: 'transparent', color: t.ink },
    onChange: (e: { target: { value: string } }) => {
      const picked = new Date(mode === 'datetime' ? e.target.value : `${e.target.value}T12:00`);
      if (!Number.isNaN(picked.getTime())) onChange({ type: 'set' }, picked);
    },
  });
}

// Only reached on Android.
export const DateTimePickerAndroid = { open: () => {} };
