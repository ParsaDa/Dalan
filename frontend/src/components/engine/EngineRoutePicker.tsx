import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Input, Select, Space } from 'antd';

import { useEngineRoutes } from '@/api/queries/useEngineRoutes';
import type { EngineRoute } from '@/schemas/engineRoute';

interface Props {
  id?: string;
  value: EngineRoute;
  onChange: (route: EngineRoute) => void;
}

const OUTBOUND_PREFIX = 'outbound:';
const GROUPS = [
  { kind: 'tunnel', label: 'pages.engineRoute.groupTunnel' },
  { kind: 'tor', label: 'pages.engineRoute.groupTor' },
  { kind: 'outbound', label: 'pages.engineRoute.groupOutbound' },
] as const;

function keyOf(route: EngineRoute): string {
  return route.via === 'outbound' ? OUTBOUND_PREFIX + route.value : route.via;
}

// Where an engine download travels: as the panel is set, straight out, through an outbound or through a link.
export default function EngineRoutePicker({ id, value, onChange }: Props) {
  const { t } = useTranslation();
  const { data: choices } = useEngineRoutes();

  const options = useMemo(() => {
    const outbounds = (choices ?? []).filter((c) => c.via === 'outbound');
    return [
      { value: 'panel', label: t('pages.engineRoute.panel') },
      { value: 'direct', label: t('pages.engineRoute.direct') },
      ...GROUPS.map((group) => ({
        label: t(group.label),
        options: outbounds
          .filter((c) => c.kind === group.kind)
          .map((c) => ({ value: OUTBOUND_PREFIX + c.value, label: c.value })),
      })).filter((group) => group.options.length > 0),
      { value: 'link', label: t('pages.engineRoute.link') },
    ];
  }, [choices, t]);

  function choose(key: string) {
    if (key.startsWith(OUTBOUND_PREFIX)) {
      onChange({ via: 'outbound', value: key.slice(OUTBOUND_PREFIX.length) });
    } else {
      onChange({ via: key as EngineRoute['via'], value: '' });
    }
  }

  return (
    <Space size={4} wrap>
      <Select
        id={id}
        size="small"
        aria-label={t('pages.engineRoute.label')}
        style={{ minWidth: 220 }}
        value={keyOf(value)}
        options={options}
        onChange={choose}
      />
      {value.via === 'link' && (
        <Input
          size="small"
          style={{ width: 280 }}
          aria-label={t('pages.engineRoute.linkLabel')}
          placeholder={t('pages.engineRoute.linkPlaceholder')}
          value={value.value}
          onChange={(e) => onChange({ via: 'link', value: e.target.value })}
        />
      )}
    </Space>
  );
}
