import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useLocation, useNavigate } from 'react-router';

import Flag from '@/components/ui/Flag';
import Icon from '../Icon';
import { Sheet } from '../overlays';
import { Pill } from '../primitives';
import { useActiveNode } from './data';
import { nodeStatus } from './useFrameStatus';

const STATUS_KEY = {
  ok: 'shell.status.online',
  warn: 'shell.status.highLoadShort',
  bad: 'shell.status.offline',
  off: 'shell.status.paused',
} as const;
const PHONE_VISIBLE = 3;

export default function NodeTabs({ phone }: { phone?: boolean }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const { direct, active, select } = useActiveNode();
  const [more, setMore] = useState(false);
  const [q, setQ] = useState('');

  const pick = (id: number | null) => {
    if (pathname !== '/') {
      navigate(id === null ? '/' : `/?node=${id}`);
      return;
    }
    select(id);
  };

  const thisTab = (
    <button
      key="this"
      type="button"
      className={`ntab${active ? '' : ' on'}`}
      aria-pressed={!active}
      onClick={() => pick(null)}
    >
      <span className="fl">
        <Icon name="server" size={16} />
      </span>
      <span>{t('pages.index.thisServer')}</span>
      <span className="sd ok" aria-hidden="true" />
    </button>
  );
  const tab = (n: (typeof direct)[number]) => {
    const st = nodeStatus(n);
    const on = active?.id === n.id;
    return (
      <button
        key={n.id}
        type="button"
        className={`ntab${on ? ' on' : ''}`}
        aria-pressed={on}
        onClick={() => pick(n.id)}
      >
        <span className="fl">
          {n.country ? <Flag country={n.country} size={24} /> : <Icon name="nodes" size={16} />}
        </span>
        <span>{n.name}</span>
        {!phone && (
          <span className="lat num">
            {st === 'bad'
              ? t('shell.status.offlineShort')
              : n.latencyMs
                ? t('shell.ms', { n: n.latencyMs })
                : ''}
          </span>
        )}
        <span className={`sd ${st}`} aria-label={t(STATUS_KEY[st])} />
      </button>
    );
  };

  if (!phone) {
    return (
      <div className="tabs" role="toolbar" aria-label={t('shell.servers')}>
        {thisTab}
        {direct.map(tab)}
        <button type="button" className="ntab add" onClick={() => navigate('/nodes?add=1')}>
          <Icon name="plus" size={16} />
          {t('shell.addNode')}
        </button>
      </div>
    );
  }

  const shown = direct.slice(0, PHONE_VISIBLE - 1);
  if (active && !shown.some((n) => n.id === active.id)) shown[shown.length - 1] = active;
  const hidden = direct.length - shown.length;
  const query = q.trim().toLowerCase();
  return (
    <>
      <div className="chiprow" role="toolbar" aria-label={t('shell.servers')}>
        {thisTab}
        {shown.map(tab)}
        {hidden > 0 && (
          <button
            type="button"
            className="ntab"
            style={{ flex: 'none', height: 36, padding: '0 14px' }}
            aria-label={t('shell.allServers', { n: direct.length + 1 })}
            onClick={() => setMore(true)}
          >
            +{hidden}
          </button>
        )}
      </div>
      <Sheet
        open={more}
        onClose={() => setMore(false)}
        label={t('shell.allServers', { n: direct.length + 1 })}
      >
        <label className="search" style={{ minWidth: 0, marginBottom: 10, cursor: 'text' }}>
          <Icon name="search" />
          <input
            aria-label={t('shell.findServer')}
            placeholder={t('shell.findServer')}
            value={q}
            onChange={(e) => setQ(e.target.value)}
            style={{
              border: 0,
              background: 'none',
              outline: 'none',
              color: 'var(--tx)',
              font: 'inherit',
              flex: 1,
              minWidth: 0,
            }}
          />
        </label>
        <div className="list">
          {direct
            .filter((n) => !query || (n.name ?? '').toLowerCase().includes(query))
            .map((n) => {
              const st = nodeStatus(n);
              return (
                <button
                  key={n.id}
                  type="button"
                  className="nav"
                  style={{ height: 54 }}
                  onClick={() => {
                    pick(n.id);
                    setMore(false);
                  }}
                >
                  <span
                    className="fl"
                    style={{
                      width: 28,
                      height: 28,
                      borderRadius: '50%',
                      overflow: 'hidden',
                      display: 'grid',
                      placeItems: 'center',
                    }}
                  >
                    {n.country ? (
                      <Flag country={n.country} size={28} />
                    ) : (
                      <Icon name="nodes" size={18} />
                    )}
                  </span>
                  <span className="f1" style={{ color: 'var(--tx)' }}>
                    {n.name}
                  </span>
                  <Pill status={st}>{t(STATUS_KEY[st])}</Pill>
                </button>
              );
            })}
        </div>
      </Sheet>
    </>
  );
}
