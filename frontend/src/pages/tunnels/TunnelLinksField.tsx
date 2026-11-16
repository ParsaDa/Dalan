import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Input, Select, Space, Typography } from 'antd';
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';

import type { TunnelOption } from '@/api/queries/useTunnels';
import { selectTagRender } from '@/components/form/SelectTag';
import { LINK_OVERRIDE_KEYS, RAKHSH_CARRIERS, isRawCarrier } from './rakhshCarriers';

// One carrier and address of a Rakhsh tunnel, plus the settings this link alone overrides.
export type LinkEntry = { carrier?: string; addr?: string } & Record<string, unknown>;

interface Props {
  id?: string;
  value?: LinkEntry[];
  onChange?: (value: LinkEntry[]) => void;
  catalog: TunnelOption[];
  renderValue: (
    option: TunnelOption,
    id: string,
    value: unknown,
    onChange: (value: unknown) => void,
  ) => ReactNode;
}

const FIXED_KEYS = ['carrier', 'addr'];

export default function TunnelLinksField({
  id,
  value = [],
  onChange,
  catalog,
  renderValue,
}: Props) {
  const { t } = useTranslation();
  const overridable = catalog.filter((option) => LINK_OVERRIDE_KEYS.includes(option.key));

  const replace = (index: number, next: LinkEntry) =>
    onChange?.(value.map((entry, n) => (n === index ? next : entry)));
  const setField = (index: number, key: string, next: unknown) =>
    replace(index, { ...value[index], [key]: next === '' ? undefined : next });
  const setOverrides = (index: number, keys: string[]) => {
    const entry = value[index];
    const kept = Object.fromEntries(
      Object.entries(entry).filter(([key]) => FIXED_KEYS.includes(key) || keys.includes(key)),
    );
    replace(index, {
      ...kept,
      ...Object.fromEntries(keys.filter((k) => !(k in kept)).map((k) => [k, undefined])),
    });
  };

  return (
    <Space orientation="vertical" size="small" style={{ width: '100%' }} id={id}>
      {value.map((entry, index) => {
        const n = index + 1;
        const chosen = Object.keys(entry).filter((key) => !FIXED_KEYS.includes(key));
        return (
          <div
            key={index}
            style={{ border: '1px solid var(--ant-color-border)', borderRadius: 6, padding: 8 }}
          >
            <Space wrap>
              <Select
                aria-label={t('pages.tunnels.linkCarrier', { n })}
                placeholder={t('pages.tunnels.transport')}
                style={{ minWidth: 140 }}
                value={entry.carrier}
                options={RAKHSH_CARRIERS}
                onChange={(carrier) => setField(index, 'carrier', carrier)}
              />
              <Input
                aria-label={t('pages.tunnels.linkAddress', { n })}
                placeholder="203.0.113.10:7000"
                autoComplete="off"
                value={entry.addr ?? ''}
                onChange={(event) => setField(index, 'addr', event.target.value)}
              />
              <Button
                type="text"
                danger
                icon={<DeleteOutlined aria-hidden />}
                aria-label={t('pages.tunnels.removeLink', { n })}
                onClick={() => onChange?.(value.filter((_, at) => at !== index))}
              />
            </Space>
            {isRawCarrier(entry.carrier) && (
              <Typography.Paragraph type="warning" style={{ margin: '4px 0 0' }}>
                {t('pages.tunnels.rakhshRawHint')}
              </Typography.Paragraph>
            )}
            {overridable.length > 0 && (
              <Select
                mode="multiple"
                allowClear
                aria-label={t('pages.tunnels.linkSettings', { n })}
                placeholder={t('pages.tunnels.linkSettingsPlaceholder')}
                style={{ width: '100%', marginTop: 8 }}
                tagRender={selectTagRender}
                value={chosen}
                options={overridable.map((option) => ({ value: option.key, label: option.key }))}
                onChange={(keys) => setOverrides(index, keys)}
              />
            )}
            {chosen.map((key) => {
              const option = overridable.find((candidate) => candidate.key === key);
              if (!option) return null;
              const fieldId = `${id ?? 'links'}-${index}-${key}`;
              return (
                <div key={key} style={{ marginTop: 8 }}>
                  <label htmlFor={fieldId}>{t('pages.tunnels.linkSetting', { n, key })}</label>
                  {renderValue(option, fieldId, entry[key], (next) => setField(index, key, next))}
                </div>
              );
            })}
          </div>
        );
      })}
      <Button
        icon={<PlusOutlined aria-hidden />}
        onClick={() => onChange?.([...value, { carrier: 'tcp', addr: '' }])}
      >
        {t('pages.tunnels.addLink')}
      </Button>
    </Space>
  );
}
