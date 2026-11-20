import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Input, Select } from 'antd';

import Flag, { FLAG_COUNTRIES } from '@/components/ui/Flag';
import AccentSwatches from '@/components/ui/AccentSwatches';
import { isCustomAccentAllowed } from '@/lib/theme/accents';
import './NodeIdentityFields.css';

interface ValueProps {
  id?: string;
  value?: string;
  onChange?: (value: string) => void;
}

const ISO2 = /^[A-Za-z]{2}$/;

function countryName(code: string, language: string): string {
  try {
    return new Intl.DisplayNames([language], { type: 'region' }).of(code) ?? code;
  } catch {
    return code;
  }
}

export function CountrySelect({ id, value, onChange }: ValueProps) {
  const { i18n } = useTranslation();
  const [typed, setTyped] = useState('');
  const options = useMemo(() => {
    const codes = [...FLAG_COUNTRIES];
    const extra = typed.toUpperCase();
    if (ISO2.test(typed) && !codes.includes(extra)) codes.push(extra);
    if (value && !codes.includes(value)) codes.push(value);
    return codes.map((code) => {
      const name = countryName(code, i18n.language);
      return {
        value: code,
        search: `${code} ${name}`.toLowerCase(),
        label: (
          <span className="node-country-option">
            <Flag country={code} size={18} />
            {name}
          </span>
        ),
      };
    });
  }, [typed, value, i18n.language]);

  return (
    <Select
      id={id}
      allowClear
      showSearch
      value={value || undefined}
      options={options}
      onSearch={setTyped}
      onChange={(next) => onChange?.((next as string | undefined) ?? '')}
      filterOption={(input, option) => option?.search.includes(input.toLowerCase()) ?? false}
    />
  );
}

export function AccentPicker({ id, value = '', onChange }: ValueProps) {
  const { t } = useTranslation();
  const isCustom = value.startsWith('#');
  const [draft, setDraft] = useState(isCustom ? value : '');
  const refused = draft.length === 7 && !isCustomAccentAllowed(draft);

  return (
    <div className="node-accent-picker" id={id} role="radiogroup">
      <button
        type="button"
        role="radio"
        aria-checked={value === ''}
        className={`node-accent-auto${value === '' ? ' on' : ''}`}
        onClick={() => onChange?.('')}
      >
        {t('pages.nodes.accentAuto')}
      </button>
      <AccentSwatches value={value} onChange={(id) => onChange?.(id)} />
      <Input
        className="node-accent-hex"
        aria-label={t('pages.nodes.accentCustom')}
        placeholder={t('pages.nodes.accentCustom')}
        maxLength={7}
        status={refused ? 'error' : undefined}
        value={draft}
        onChange={(event) => {
          const next = event.target.value.trim().toLowerCase();
          setDraft(next);
          if (isCustomAccentAllowed(next)) onChange?.(next);
        }}
      />
      {refused && <div className="node-accent-refused">{t('pages.nodes.accentCustomRefused')}</div>}
    </div>
  );
}
