import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useQuery } from '@tanstack/react-query';

import Icon from '@/components/dl/Icon';
import { DropChip } from '@/components/dl/DropChip';
import { Btn, Chip, Empty as DlEmpty, Sk } from '@/components/dl/primitives';
import { fmtInt } from '@/components/dl/numbers';
import { parseLogLine } from '@/pages/index/logParse';
import { FileManager, HttpUtil } from '@/utils';

type Source = 'panel' | 'xray';
type Level = 'info' | 'warning' | 'err';

interface XrayLogEntry {
  DateTime?: string | number;
  FromAddress?: string;
  ToAddress?: string;
  Inbound?: string;
  Outbound?: string;
  Email?: string;
  Event?: number;
}

export interface LogLine {
  time: string;
  level: 'info' | 'warn' | 'error' | 'debug';
  source: string;
  text: string;
}

const STREAM_MS = 4000;
const EVENTS: Record<number, string> = { 0: 'direct', 1: 'blocked', 2: 'proxy' };

// One panel log line in the page's shape: time, level, where it came from and the message.
export function panelLine(raw: string): LogLine {
  const p = parseLogLine(raw);
  const lv = p.levelText.toUpperCase();
  return {
    time: p.time || p.stamp,
    level: lv === 'ERROR' ? 'error' : lv === 'WARNING' ? 'warn' : lv === 'DEBUG' ? 'debug' : 'info',
    source: p.service === 'XRAY:' ? 'Xray' : 'Panel',
    text: p.body,
  };
}

// An Xray access-log entry as a line, phrased like Xray's own access log.
export function xrayLine(e: XrayLogEntry): LogLine {
  const at =
    typeof e.DateTime === 'number' ? new Date(e.DateTime * 1000) : new Date(e.DateTime || 0);
  const time = Number.isNaN(at.getTime()) ? '' : at.toTimeString().slice(0, 8);
  const route = [e.Inbound, e.Outbound].filter(Boolean).join(' >> ');
  const text = [
    e.Event === 1 ? 'rejected' : 'accepted',
    e.ToAddress || '',
    e.FromAddress ? `from ${e.FromAddress}` : '',
    route ? `[${route}]` : '',
    e.Email ? `email: ${e.Email}` : '',
  ]
    .filter(Boolean)
    .join(' ');
  return {
    time,
    level: e.Event === 1 ? 'warn' : 'info',
    source: `Xray · ${EVENTS[e.Event ?? 0] ?? ''}`,
    text,
  };
}

async function fetchLogs(source: Source, rows: number, level: Level): Promise<LogLine[]> {
  if (source === 'xray') {
    const msg = await HttpUtil.post<XrayLogEntry[]>(
      `/panel/api/server/xraylogs/${rows}`,
      { filter: '', showDirect: true, showBlocked: true, showProxy: true },
      { silent: true },
    );
    if (!msg?.success) throw new Error(msg?.msg || 'xray logs');
    return (msg.obj || []).map(xrayLine).reverse();
  }
  const msg = await HttpUtil.post<string[]>(
    `/panel/api/server/logs/${rows}`,
    { level, syslog: false },
    { silent: true },
  );
  if (!msg?.success) throw new Error(msg?.msg || 'logs');
  // The panel sends its newest line first already; the Xray access log comes oldest first.
  return (msg.obj || []).map(panelLine);
}

const PILL: Record<LogLine['level'], string> = {
  info: 'neutral',
  debug: 'neutral',
  warn: 'warn',
  error: 'bad',
};

export default function LogsPage() {
  const { t, i18n } = useTranslation();
  const [source, setSource] = useState<Source>('panel');
  const [level, setLevel] = useState<Level>('info');
  const [rows, setRows] = useState('100');
  const [paused, setPaused] = useState(false);
  const [search, setSearch] = useState('');
  const query = useQuery({
    queryKey: ['logs', source, rows, level],
    queryFn: () => fetchLogs(source, Number(rows), level),
    refetchInterval: paused ? false : STREAM_MS,
    retry: false,
  });

  const lines = useMemo(() => {
    const q = search.trim().toLowerCase();
    const all = query.data ?? [];
    return q ? all.filter((l) => `${l.source} ${l.text}`.toLowerCase().includes(q)) : all;
  }, [query.data, search]);

  const download = () =>
    FileManager.downloadTextFile(
      (query.data ?? [])
        .map((l) => `${l.time} ${l.level.toUpperCase()} ${l.source} ${l.text}`)
        .join('\n'),
      `${source}-logs.txt`,
    );

  return (
    <div className="col stag" style={{ gap: 18 }}>
      <div className="pagehead">
        <div>
          <h2 className="h2" style={{ fontSize: 28 }}>
            {t('menu.logs')}
          </h2>
          <p>{t('logs.subtitle')}</p>
        </div>
        <div className="row wrap">
          <Btn icon="down" disabled={!query.data?.length} onClick={download}>
            {t('download')}
          </Btn>
          <Btn icon="power" aria-pressed={paused} onClick={() => setPaused(!paused)}>
            {paused ? t('logs.resume') : t('logs.pause')}
          </Btn>
        </div>
      </div>
      <div className="row wrap sp">
        <div className="chips" role="group" aria-label={t('logs.levels')}>
          {(['info', 'warning', 'err'] as const).map((lv) => (
            <Chip key={lv} on={level === lv} onClick={() => setLevel(lv)}>
              {t(`logs.level.${lv}`)}
            </Chip>
          ))}
        </div>
        <div className="row wrap">
          <div className="chips" role="group" aria-label={t('logs.sources')}>
            {(['panel', 'xray'] as const).map((s) => (
              <Chip key={s} on={source === s} onClick={() => setSource(s)}>
                {t(`logs.source.${s}`)}
              </Chip>
            ))}
          </div>
          <DropChip
            label={t('pages.index.logLines')}
            value={rows}
            onChange={setRows}
            options={['50', '100', '200', '500'].map((n) => ({ value: n, label: n }))}
          />
          <label className="search" style={{ minWidth: 240, cursor: 'text' }}>
            <Icon name="search" />
            <input
              aria-label={t('logs.search')}
              placeholder={t('logs.search')}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
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
        </div>
      </div>
      <section className="tile dense" style={{ padding: 0 }}>
        <div
          className="row"
          style={{ padding: '10px 16px', borderBottom: '1px solid var(--glass-b)', gap: 8 }}
        >
          {paused ? <Icon name="power" size={14} /> : <span className="live" aria-hidden="true" />}
          <b>{paused ? t('logs.paused') : t('logs.streaming')}</b>
          <span className="mut" style={{ fontSize: 12.5 }}>
            {t('logs.count', {
              n: fmtInt(lines.length),
              at: query.dataUpdatedAt
                ? new Date(query.dataUpdatedAt).toLocaleTimeString(i18n.language)
                : '–',
            })}
          </span>
        </div>
        {query.isPending ? (
          <div style={{ padding: 16 }}>
            <Sk h={200} />
          </div>
        ) : query.isError ? (
          <DlEmpty title={t('logs.unavailable')}>
            {source === 'xray' ? t('logs.xrayHint') : String(query.error.message)}
          </DlEmpty>
        ) : lines.length === 0 ? (
          <DlEmpty title={t('logs.empty')} />
        ) : (
          <div
            className="code"
            role="log"
            aria-live="off"
            style={{
              border: 0,
              background: 'transparent',
              borderRadius: 0,
              fontSize: 12.5,
              maxHeight: 'calc(100vh - 360px)',
              overflow: 'auto',
            }}
          >
            {lines.map((l, i) => (
              <div key={i} className="ln logln" style={{ gap: 14, padding: '3px 16px' }}>
                <span className="mut">{l.time}</span>
                <span
                  className={`pill ${PILL[l.level]}`}
                  style={{ height: 20, minWidth: 62, justifyContent: 'center' }}
                >
                  {l.level}
                </span>
                <span className="badge" style={{ minWidth: 74 }}>
                  {l.source}
                </span>
                <span
                  style={{ whiteSpace: 'pre-wrap', color: 'var(--tx)', overflowWrap: 'anywhere' }}
                >
                  {l.text}
                </span>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
