import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import i18next from 'i18next';

// Persian pages show Persian digits; every other locale keeps Latin digits.
export function numberLocale(lang: string): string {
  return lang.startsWith('fa') ? 'fa-IR' : 'en-US';
}

export function useNum() {
  const { i18n } = useTranslation();
  const loc = numberLocale(i18n.language || 'en-US');
  return useMemo(() => {
    const ints = new Intl.NumberFormat(loc, { maximumFractionDigits: 0 });
    const pcts = new Intl.NumberFormat(loc, { style: 'percent', maximumFractionDigits: 0 });
    return {
      int: (n: number) => ints.format(n),
      pct: (v: number) => pcts.format(v),
      dec: (n: number, digits = 1) =>
        new Intl.NumberFormat(loc, {
          minimumFractionDigits: digits,
          maximumFractionDigits: digits,
        }).format(n),
    };
  }, [loc]);
}

export function fmtInt(n: number): string {
  return new Intl.NumberFormat(numberLocale(i18next.language || 'en-US'), {
    maximumFractionDigits: 0,
  }).format(n);
}
