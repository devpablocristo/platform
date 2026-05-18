import { useEffect, useState, type InputHTMLAttributes } from 'react';
import {
  formatSchedulingDateOnly,
  parseSchedulingDateOnlyLatinInput,
  resolveSchedulingCopyLocale,
} from './locale';

export type SchedulingDateInputProps = Omit<
  InputHTMLAttributes<HTMLInputElement>,
  'type' | 'value' | 'onChange'
> & {
  value: string;
  onValueChange: (isoYmd: string) => void;
  locale?: string;
};

function displayLocaleForScheduling(locale: string | undefined): string {
  return resolveSchedulingCopyLocale(locale) === 'es' ? (locale ?? 'es-AR') : (locale ?? 'en-US');
}

export function SchedulingDateInput({
  value,
  onValueChange,
  locale,
  ...rest
}: SchedulingDateInputProps) {
  const isEs = resolveSchedulingCopyLocale(locale) === 'es';
  const fmtLocale = displayLocaleForScheduling(locale);
  const [text, setText] = useState(() =>
    isEs ? formatSchedulingDateOnly(value, fmtLocale) : value,
  );

  useEffect(() => {
    if (isEs) {
      setText(formatSchedulingDateOnly(value, fmtLocale));
    }
  }, [value, isEs, fmtLocale]);

  if (!isEs) {
    return (
      <input
        {...rest}
        type="date"
        value={value}
        onChange={(e) => onValueChange(e.target.value)}
      />
    );
  }

  return (
    <input
      {...rest}
      type="text"
      inputMode="numeric"
      autoComplete="off"
      placeholder="DD/MM/AAAA"
      lang="es"
      value={text}
      onChange={(e) => setText(e.target.value)}
      onBlur={() => {
        const iso = parseSchedulingDateOnlyLatinInput(text);
        if (iso) {
          onValueChange(iso);
          setText(formatSchedulingDateOnly(iso, fmtLocale));
        } else {
          setText(formatSchedulingDateOnly(value, fmtLocale));
        }
      }}
    />
  );
}
