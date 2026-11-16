import { useTranslation } from 'react-i18next';
import { Collapse, Form, Input, InputNumber, Select, Space, Tag, Typography } from 'antd';

import { selectTagRender } from '@/components/form/SelectTag';
import type { TunnelOption, TunnelOptionValues } from '@/api/queries/useTunnels';

// Setting a value is optional: an empty field is left out so the engine keeps its own default.
export function cleanOptions(values: TunnelOptionValues | undefined): TunnelOptionValues {
  return Object.fromEntries(
    Object.entries(values ?? {}).filter(([, v]) => {
      if (v === undefined || v === null || v === '') return false;
      return !(Array.isArray(v) && v.length === 0);
    }),
  );
}

// Form.Item hands the id, value and onChange to its direct child, so they are passed on to the real control.
interface ControlProps<V> {
  id?: string;
  value?: V;
  onChange?: (value: V) => void;
}

function OptionControl({ option, ...control }: { option: TunnelOption } & ControlProps<never>) {
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
}

// Every BackPack setting for the mode, grouped as the engine's catalog groups them.
export default function TunnelOptionFields({ options, mode, skip = [] }: Props) {
  const { t } = useTranslation();
  const groups = new Map<string, TunnelOption[]>();
  for (const option of options) {
    if (option.mode !== mode || skip.includes(option.key)) continue;
    groups.set(option.group, [...(groups.get(option.group) ?? []), option]);
  }
  if (groups.size === 0) return null;
  return (
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
  );
}
