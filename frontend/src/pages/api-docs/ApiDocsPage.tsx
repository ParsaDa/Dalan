import { useState } from 'react';
import { useNavigate } from 'react-router';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Tabs } from 'antd';
import SwaggerUI from 'swagger-ui-react';
import 'swagger-ui-react/swagger-ui.css';

import Icon from '@/components/dl/Icon';
import { Btn, Pill, Seg, Sk, TileHead } from '@/components/dl/primitives';
import { HttpUtil } from '@/utils';
import { useTheme } from '@/hooks/useTheme';
import { EXAMPLES } from '@/generated/examples';
import { buildWebSocketEvents } from './websocket-events';
import './ApiDocsPage.css';

const basePath = window.X_UI_BASE_PATH || '';
const openApiUrl = `${basePath}panel/api/openapi.json`;
const websocketEvents = buildWebSocketEvents(EXAMPLES);

interface TaggedOperations {
  keySeq: () => { first: () => string | undefined };
  filter: (keep: (operations: unknown, tag: string) => boolean) => TaggedOperations;
}

interface LayoutSelectors {
  currentFilter: () => string | false;
}

interface SectionTabsProps {
  specSelectors: { tags: () => { toJS: () => { name: string }[] } };
  layoutSelectors: LayoutSelectors;
  layoutActions: { updateFilter: (tag: string) => void };
}

function SectionTabs({ specSelectors, layoutSelectors, layoutActions }: SectionTabsProps) {
  const tags = specSelectors
    .tags()
    .toJS()
    .map((tag) => tag.name);
  return (
    <div className="wrapper section-tabs">
      <Tabs
        size="small"
        activeKey={layoutSelectors.currentFilter() || tags[0]}
        onChange={layoutActions.updateFilter}
        items={tags.map((tag) => ({ key: tag, label: tag }))}
      />
    </div>
  );
}

// Shows one tag at a time, the first until a tab is picked. Swagger's own filter is a
// substring match ("Settings" would also show "Xray Settings") and no-op while unset.
const sectionTabsPlugin = {
  statePlugins: {
    spec: {
      wrapSelectors: {
        taggedOperations:
          (
            select: (...args: unknown[]) => TaggedOperations,
            system: { getSystem: () => { layoutSelectors: LayoutSelectors } },
          ) =>
          (...args: unknown[]) => {
            const operations = select(...args);
            const active =
              system.getSystem().layoutSelectors.currentFilter() || operations.keySeq().first();
            return operations.filter((_, tag) => tag === active);
          },
      },
    },
  },
  components: { FilterContainer: SectionTabs },
};

interface TokenRow {
  id: number;
  name: string;
  enabled: boolean;
  scope: string;
  expiresAt: number;
}

function TokensTile() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { data, isPending } = useQuery({
    queryKey: ['settings', 'apiTokens'],
    queryFn: async () => {
      const msg = await HttpUtil.get<TokenRow[]>('/panel/api/setting/apiTokens', undefined, {
        silent: true,
      });
      return msg?.success ? (msg.obj ?? []) : [];
    },
  });
  return (
    <section className="tile s5" style={{ alignSelf: 'start' }}>
      <TileHead
        icon="key"
        title={t('developer.tokens')}
        right={
          <Btn size="sm" kind="pri" icon="plus" onClick={() => navigate('/settings#security')}>
            {t('developer.newToken')}
          </Btn>
        }
      />
      {isPending ? (
        <Sk h={80} />
      ) : !data?.length ? (
        <div className="mut" style={{ fontSize: 13 }}>
          {t('developer.noTokens')}
        </div>
      ) : (
        <div className="list">
          {data.map((tok) => (
            <div className="li" key={tok.id}>
              <span className={`ico${tok.enabled ? ' acc' : ''}`}>
                <Icon name="key" />
              </span>
              <div className="f1" style={{ minWidth: 0 }}>
                <b>{tok.name}</b>
                <div className="mut mono" style={{ fontSize: 12 }}>
                  {tok.scope}
                  {tok.expiresAt > 0
                    ? ` · ${new Date(tok.expiresAt * (tok.expiresAt < 1e11 ? 1000 : 1)).toLocaleDateString()}`
                    : ''}
                </div>
              </div>
              <Pill status={tok.enabled ? 'ok' : 'off'}>
                {tok.enabled ? t('enabled') : t('disabled')}
              </Pill>
            </div>
          ))}
        </div>
      )}
      <div className="sep" style={{ margin: '12px 0' }} />
      <Btn size="sm" icon="lock" onClick={() => navigate('/settings#security')}>
        {t('developer.manage')}
      </Btn>
    </section>
  );
}

export default function ApiDocsPage() {
  const { t } = useTranslation();
  const [view, setView] = useState<'api' | 'ws'>('api');
  const { isDark, isUltra } = useTheme();
  const [loaded, setLoaded] = useState(false);
  return (
    <div className="col stag" style={{ gap: 18 }}>
      <div className="pagehead">
        <div>
          <div className="crumb">
            {t('menu.settings')}
            <Icon name="chev" className="flipx" />
            <span style={{ color: 'var(--tx)' }}>{t('settingsPage.nav.developer')}</span>
          </div>
          <h2 className="h2" style={{ fontSize: 28, marginTop: 6 }}>
            {t('settingsPage.nav.developer')}
          </h2>
          <p>{t('settingsPage.developerHint')}</p>
        </div>
      </div>
      <div className="grid">
        <TokensTile />
        <section
          className={`tile s7 api-docs-page${isDark ? ' is-dark' : ''}${isUltra ? ' is-ultra' : ''}`}
          style={{ alignSelf: 'start' }}
        >
          <TileHead
            icon="code"
            title={t('developer.reference')}
            right={
              <Seg
                label={t('developer.reference')}
                value={view}
                onChange={setView}
                options={[
                  { value: 'api', label: t('developer.panelApi') },
                  { value: 'ws', label: t('developer.wsEvents') },
                ]}
              />
            }
          />
          {view === 'api' ? (
            loaded ? (
              <div className="docs-wrapper" role="region" aria-label={t('menu.apiDocs')}>
                <SwaggerUI
                  url={openApiUrl}
                  docExpansion="list"
                  deepLinking={false}
                  plugins={[sectionTabsPlugin]}
                  tryItOutEnabled
                  persistAuthorization
                />
              </div>
            ) : (
              <div className="col" style={{ gap: 10, alignItems: 'flex-start' }}>
                <span className="pill neutral">{t('developer.onDemand')}</span>
                <Btn kind="pri" icon="code" onClick={() => setLoaded(true)}>
                  {t('developer.load')}
                </Btn>
              </div>
            )
          ) : (
            <section className="websocket-events col" style={{ gap: 12 }}>
              <p className="mut" style={{ margin: 0 }}>
                {t('developer.wsIntro')}
              </p>
              {websocketEvents.map((event) => (
                <div key={event.type} className="tile" style={{ padding: 14 }}>
                  <b className="mono">{event.type}</b>
                  <p className="mut" style={{ margin: '4px 0 8px', fontSize: 13 }}>
                    {event.summary}
                  </p>
                  <pre
                    className="code mono"
                    style={{ margin: 0, padding: 10, fontSize: 12, overflow: 'auto' }}
                  >
                    {JSON.stringify(event.example, null, 2)}
                  </pre>
                </div>
              ))}
            </section>
          )}
        </section>
      </div>
    </div>
  );
}
