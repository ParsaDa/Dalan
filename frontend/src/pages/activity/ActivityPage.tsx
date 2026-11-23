import { useState } from 'react';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { useActivity } from '@/api/queries/useDashboard';
import type { ActivityParams } from '@/api/queries/useDashboard';
import Icon from '@/components/dl/Icon';
import type { IconName } from '@/components/dl/Icon';
import { Badge, Btn, Chip, Empty, Sk } from '@/components/dl/primitives';
import { Modal } from '@/components/dl/overlays';
import { DropChip } from '@/components/dl/DropChip';
import { fmtInt } from '@/components/dl/numbers';
import { useActiveNode, useCurrentUser } from '@/components/dl/shell/data';
import { IlloEmpty } from '@/components/dl/Illustrations';
import type { Activity } from '@/generated/zod';
import { activityIcon, activitySentence, relTime } from '@/pages/index/dashboard/ActivityList';
import { HttpUtil, FileManager } from '@/utils';

const PAGE_SIZE = 25;
type Area = '' | 'client' | 'tunnel' | 'node' | 'session' | 'system';
const AREAS: Area[] = ['', 'client', 'tunnel', 'node', 'session', 'system'];

function csvCell(v: string | number): string {
  const s = String(v);
  return /[",\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
}

function WhoCell({ name, sub }: { name: string; sub: string }) {
  return (
    <div className="who">
      <i>{name.slice(0, 2).toUpperCase()}</i>
      <div>
        <div style={{ fontWeight: 750 }}>{name}</div>
        <div className="mut" style={{ fontSize: 12 }}>
          {sub}
        </div>
      </div>
    </div>
  );
}

function WhatCell({ icon, children }: { icon: IconName; children: ReactNode }) {
  return (
    <div className="row" style={{ gap: 8 }}>
      <span className="ico acc" style={{ width: 28, height: 28 }}>
        <Icon name={icon} size={15} />
      </span>
      <span>{children}</span>
    </div>
  );
}

export default function ActivityPage() {
  const { t, i18n } = useTranslation();
  const user = useCurrentUser();
  const { nodes } = useActiveNode();
  const [area, setArea] = useState<Area>('');
  const [actor, setActor] = useState('');
  const [node, setNode] = useState('');
  const [days, setDays] = useState('7');
  const [page, setPage] = useState(1);
  const [detail, setDetail] = useState<Activity | null>(null);
  const params: ActivityParams = {
    page,
    pageSize: PAGE_SIZE,
    category: area && area !== 'system' ? area : undefined,
    actor: area === 'system' ? 'system' : actor || undefined,
    nodeId: node === '' ? undefined : Number(node),
    days: days === 'all' ? undefined : Number(days),
  };
  const { items, total, loaded } = useActivity(params);
  const pages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const refs = nodes.map((n) => ({ id: n.id, name: n.name ?? `#${n.id}`, country: n.country }));
  const where = (a: Activity) =>
    a.nodeId
      ? (refs.find((n) => n.id === a.nodeId)?.name ?? `#${a.nodeId}`)
      : t('pages.index.thisServer');
  const who = (a: Activity) =>
    a.actor === 'you'
      ? user || t('dashboard.activity.who.you')
      : t(`dashboard.activity.who.${a.actor}`, { defaultValue: a.actor });
  const pick = (setter: (v: string) => void) => (v: string) => {
    setter(v);
    setPage(1);
  };

  const exportCsv = async () => {
    const rows: Activity[] = [];
    for (let p = 1; p <= Math.ceil(total / 200); p++) {
      const msg = await HttpUtil.get<{ items: Activity[] }>('/panel/api/activity/list', {
        ...params,
        page: p,
        pageSize: 200,
      });
      if (!msg?.success || !msg.obj) break;
      rows.push(...msg.obj.items);
    }
    const head = ['time', 'actor', 'action', 'target', 'server'];
    const body = rows.map((a) =>
      [new Date(a.at * 1000).toISOString(), a.actor, a.action, a.target, where(a)]
        .map(csvCell)
        .join(','),
    );
    FileManager.downloadTextFile([head.join(','), ...body].join('\n'), 'dalan-activity.csv', {
      type: 'text/csv',
    });
  };

  return (
    <div className="col stag" style={{ gap: 18 }}>
      <div className="pagehead">
        <div>
          <h2 className="h2" style={{ fontSize: 28 }}>
            {t('menu.activity')}
          </h2>
          <p>{t('activity.sub')}</p>
        </div>
        <div className="row wrap">
          <Btn icon="down" onClick={exportCsv} disabled={!total}>
            {t('activity.export')}
          </Btn>
        </div>
      </div>
      <div className="row wrap" style={{ gap: 8 }}>
        <div className="chips" role="group" aria-label={t('activity.filters')}>
          {AREAS.map((a) => (
            <Chip
              key={a || 'all'}
              on={area === a}
              onClick={() => pick((v) => setArea(v as Area))(a)}
            >
              {t(`activity.area.${a || 'all'}`)}
            </Chip>
          ))}
        </div>
        <DropChip
          icon="nodes"
          label={t('activity.server')}
          value={node}
          onChange={pick(setNode)}
          options={[
            { value: '', label: t('activity.any') },
            { value: '0', label: t('pages.index.thisServer') },
            ...refs.map((n) => ({ value: String(n.id), label: n.name })),
          ]}
        />
        {area !== 'system' && (
          <DropChip
            icon="users"
            label={t('activity.who')}
            value={actor}
            onChange={pick(setActor)}
            options={[
              { value: '', label: t('activity.anyone') },
              { value: 'you', label: user || t('dashboard.activity.who.you') },
              { value: 'api', label: t('dashboard.activity.who.api') },
              { value: 'system', label: t('dashboard.activity.who.system') },
            ]}
          />
        )}
        <DropChip
          icon="clock"
          label={t('activity.when')}
          value={days}
          onChange={pick(setDays)}
          options={[
            { value: '1', label: t('activity.days', { n: fmtInt(1) }) },
            { value: '7', label: t('activity.days', { n: fmtInt(7) }) },
            { value: '30', label: t('activity.days', { n: fmtInt(30) }) },
            { value: 'all', label: t('activity.allTime') },
          ]}
        />
      </div>
      <section className="tile dense" style={{ padding: 20 }}>
        {!loaded ? (
          <div className="col" style={{ gap: 12 }}>
            {[70, 55, 80, 62, 48].map((w) => (
              <div key={w} className="row" style={{ gap: 12 }}>
                <Sk w={32} h={32} r={10} />
                <div className="f1 col" style={{ gap: 6 }}>
                  <Sk w={`${w}%`} h={11} />
                  <Sk w={`${w - 20}%`} h={9} />
                </div>
              </div>
            ))}
          </div>
        ) : items.length === 0 ? (
          <Empty art={<IlloEmpty size={130} />} title={t('activity.emptyTitle')}>
            {t('activity.emptyBody')}
          </Empty>
        ) : (
          <>
            <div className="list only-narrow">
              {items.map((a) => (
                <button
                  type="button"
                  key={a.id}
                  className="li"
                  onClick={() => setDetail(a)}
                  style={{
                    background: 'none',
                    border: 0,
                    color: 'inherit',
                    font: 'inherit',
                    textAlign: 'start',
                    width: '100%',
                    cursor: 'pointer',
                    alignItems: 'flex-start',
                  }}
                >
                  <span className="ico acc">
                    <Icon name={activityIcon(a.action)} />
                  </span>
                  <span className="f1 col" style={{ gap: 4 }}>
                    <span>{activitySentence(t, (k) => i18n.exists(k), a)}</span>
                    <span className="mut" style={{ fontSize: 12.5, fontWeight: 650 }}>
                      {who(a)} · {where(a)} · {relTime(t, a.at)}
                    </span>
                  </span>
                </button>
              ))}
            </div>
            <div className="tblbox only-wide">
              <table className="tbl" style={{ minWidth: 820 }}>
                <thead>
                  <tr>
                    <th>{t('activity.col.when')}</th>
                    <th>{t('activity.col.who')}</th>
                    <th>{t('activity.col.what')}</th>
                    <th>{t('activity.col.where')}</th>
                    <th>
                      <span
                        style={{ position: 'absolute', width: 1, height: 1, overflow: 'hidden' }}
                      >
                        {t('activity.col.details')}
                      </span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {items.map((a) => (
                    <tr key={a.id}>
                      <td>
                        <span className="num mut" style={{ fontWeight: 650 }}>
                          {relTime(t, a.at)}
                        </span>
                      </td>
                      <td>
                        <WhoCell
                          name={who(a)}
                          sub={t(`activity.role.${a.actor}`, { defaultValue: '' })}
                        />
                      </td>
                      <td style={{ whiteSpace: 'normal' }}>
                        <WhatCell icon={activityIcon(a.action)}>
                          {activitySentence(t, (k) => i18n.exists(k), a)}
                        </WhatCell>
                      </td>
                      <td>
                        <Badge>{where(a)}</Badge>
                        {a.nodeId > 0 && a.actor !== 'system' && (
                          <>
                            {' '}
                            <Badge acc>{t('activity.throughPanel')}</Badge>
                          </>
                        )}
                      </td>
                      <td>
                        <div className="row" style={{ justifyContent: 'flex-end' }}>
                          <button
                            type="button"
                            className="ib sm"
                            aria-label={t('activity.col.details')}
                            onClick={() => setDetail(a)}
                          >
                            <Icon name="eye" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </>
        )}
      </section>
      {pages > 1 && (
        <div className="row sp mut" style={{ fontWeight: 650, fontSize: 13 }}>
          <span>
            {t('activity.showing', {
              from: fmtInt((page - 1) * PAGE_SIZE + 1),
              to: fmtInt(Math.min(total, page * PAGE_SIZE)),
              total: fmtInt(total),
            })}
          </span>
          <div className="row" style={{ gap: 6 }}>
            <Btn
              size="sm"
              icon="back"
              aria-label={t('activity.prev')}
              disabled={page <= 1}
              onClick={() => setPage(page - 1)}
            />
            <Badge acc>{fmtInt(page)}</Badge>
            <span>/ {fmtInt(pages)}</span>
            <Btn
              size="sm"
              icon="chev"
              aria-label={t('activity.next')}
              disabled={page >= pages}
              onClick={() => setPage(page + 1)}
            />
          </div>
        </div>
      )}
      <Modal
        open={detail !== null}
        onClose={() => setDetail(null)}
        label={t('activity.col.details')}
        width={460}
      >
        {detail && (
          <>
            <div className="row sp">
              <h2 className="h2">{t('activity.col.details')}</h2>
              <button
                type="button"
                className="ib sm"
                aria-label={t('close')}
                onClick={() => setDetail(null)}
              >
                <Icon name="x" />
              </button>
            </div>
            <p style={{ margin: 0 }}>{activitySentence(t, (k) => i18n.exists(k), detail)}</p>
            <dl className="kv">
              <dt>{t('activity.col.when')}</dt>
              <dd className="num">{new Date(detail.at * 1000).toLocaleString(i18n.language)}</dd>
              <dt>{t('activity.col.who')}</dt>
              <dd>{who(detail)}</dd>
              <dt>{t('activity.col.where')}</dt>
              <dd>{where(detail)}</dd>
              <dt>{t('activity.code')}</dt>
              <dd className="mono">{detail.action}</dd>
            </dl>
          </>
        )}
      </Modal>
    </div>
  );
}
