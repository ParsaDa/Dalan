import { createContext, useContext } from 'react';
import { useTranslation } from 'react-i18next';
import { Collapse, Form, Input, InputNumber, Select, Space, Tag, Typography } from 'antd';

import { selectTagRender } from '@/components/form/SelectTag';
import type { TunnelOption, TunnelOptionValues } from '@/api/queries/useTunnels';
import TunnelLinksField, { type LinkEntry } from './TunnelLinksField';

// The links control needs the whole catalog to offer the settings one link may override.
const CatalogContext = createContext<TunnelOption[]>([]);
export const useOptionCatalog = () => useContext(CatalogContext);

// Setting a value is optional: an empty field is left out so the engine keeps its own default.
export function cleanOptions(values: TunnelOptionValues | undefined): TunnelOptionValues {
  return Object.fromEntries(
    Object.entries(values ?? {})
      .filter(([, v]) => {
        if (v === undefined || v === null || v === '') return false;
        return !(Array.isArray(v) && v.length === 0);
      })
      .map(([key, v]) => [key, key === 'links' ? cleanLinks(v) : v]),
  );
}

// A link keeps only the overrides that were filled in, so the engine inherits the rest.
function cleanLinks(links: unknown) {
  if (!Array.isArray(links)) return links;
  return links.map((entry: LinkEntry) =>
    Object.fromEntries(Object.entries(entry).filter(([, v]) => v !== undefined && v !== '')),
  );
}

// Form.Item hands the id, value and onChange to its direct child, so they are passed on to the real control.
interface ControlProps<V> {
  id?: string;
  value?: V;
  onChange?: (value: V) => void;
}

function LinksControl(control: ControlProps<LinkEntry[]>) {
  const catalog = useOptionCatalog();
  return (
    <TunnelLinksField
      {...control}
      catalog={catalog}
      renderValue={(option, id, value, onChange) => (
        <OptionControl option={option} id={id} value={value as never} onChange={onChange} />
      )}
    />
  );
}

export function OptionControl({
  option,
  ...control
}: { option: TunnelOption } & ControlProps<never>) {
  const { t } = useTranslation();
  switch (option.kind) {
    case 'bool':
      return (
        <Select
          {...control}
          allowClear
          placeholder={t('pages.tunnels.optionDefault')}
          options={[
            { value: true, label: t('pages.tunnels.optionOn') },
            { value: false, label: t('pages.tunnels.optionOff') },
          ]}
        />
      );
    case 'int':
      return (
        <InputNumber<number>
          id={control.id}
          value={control.value}
          onChange={(v) => control.onChange?.((v ?? undefined) as never)}
          min={option.min}
          max={option.max || undefined}
          placeholder={t('pages.tunnels.optionDefault')}
          style={{ width: '100%' }}
        />
      );
    case 'choice':
      return (
        <Select
          {...control}
          allowClear
          placeholder={t('pages.tunnels.optionDefault')}
          options={(option.choices ?? []).map((value) => ({ value, label: value }))}
        />
      );
    case 'links':
      return (
        <LinksControl
          id={control.id}
          value={control.value as LinkEntry[] | undefined}
          onChange={control.onChange as ((value: LinkEntry[]) => void) | undefined}
        />
      );
    case 'list':
      return (
        <Select
          {...control}
          mode="tags"
          open={false}
          tagRender={selectTagRender}
          tokenSeparators={[' ', ',']}
          placeholder={t('pages.tunnels.optionDefault')}
        />
      );
    default:
      return (
        <Input {...control} autoComplete="off" placeholder={t('pages.tunnels.optionDefault')} />
      );
  }
}

export function TunnelOptionField({
  option,
  required,
}: {
  option: TunnelOption;
  required?: boolean;
}) {
  const { t, i18n } = useTranslation();
  const hint = i18n.language.startsWith('fa') ? option.hintFa : option.hintEn;
  return (
    <Form.Item
      name={['options', option.key]}
      label={
        <Space size={4}>
          <Typography.Text code>{option.key}</Typography.Text>
          {option.role !== 'both' && (
            <Tag>
              {option.role === 'portal'
                ? t('pages.tunnels.rolePortal')
                : t('pages.tunnels.roleBridge')}
            </Tag>
          )}
        </Space>
      }
      extra={hint}
      rules={
        required ? [{ required: true, message: t('pages.tunnels.optionRequired') }] : undefined
      }
    >
      <OptionControl option={option} />
    </Form.Item>
  );
}

interface Props {
  options: TunnelOption[];
  mode: 'reverse' | 'direct';
  // Keys the form handles itself (the carrier) or the panel fills in (the link addresses).
  skip?: string[];
  // Keys shown up front and required, because the chosen carrier cannot work without them.
  pinned?: string[];
  // Hides settings only the other end reads; BackPack's catalog is shown whole, so it passes none.
  forRole?: 'portal' | 'bridge';
}

// Every setting of the engine for the mode, grouped as the engine's catalog groups them.
export default function TunnelOptionFields({
  options,
  mode,
  skip = [],
  pinned = [],
  forRole,
}: Props) {
  const { t } = useTranslation();
  const groups = new Map<string, TunnelOption[]>();
  for (const option of options) {
    const otherMode = option.mode !== 'both' && option.mode !== mode;
    const otherRole = forRole !== undefined && option.role !== 'both' && option.role !== forRole;
    if (otherMode || otherRole || skip.includes(option.key) || pinned.includes(option.key))
      continue;
    groups.set(option.group, [...(groups.get(option.group) ?? []), option]);
  }
  const pinnedOptions = options.filter((option) => pinned.includes(option.key));
  if (groups.size === 0 && pinnedOptions.length === 0) return null;
  return (
    <CatalogContext.Provider value={options}>
      {pinnedOptions.map((option) => (
        <TunnelOptionField key={option.key} option={option} required />
      ))}
      {groups.size > 0 && (
        <Collapse
          size="small"
          style={{ marginBottom: 16 }}
          items={[
            {
              key: 'advanced',
              label: t('pages.tunnels.advancedOptions'),
              children: (
                <Collapse
                  size="small"
                  items={[...groups].map(([group, items]) => ({
                    key: group,
                    label: t(`pages.tunnels.optionGroups.${group}`),
                    forceRender: true,
                    children: items.map((option) => (
                      <TunnelOptionField key={option.key} option={option} />
                    )),
                  }))}
                />
              ),
              forceRender: true,
            },
          ]}
        />
      )}
    </CatalogContext.Provider>
  );
}
