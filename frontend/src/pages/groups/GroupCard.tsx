import { useTranslation } from 'react-i18next';

import Icon from '@/components/dl/Icon';
import { MoreMenu, type MenuAction } from '@/components/dl/MoreMenu';
import { IconBtn, Meter } from '@/components/dl/primitives';
import { useNum } from '@/components/dl/numbers';
import { accentScope } from '@/components/dl/accentScope';
import { useTheme } from '@/hooks/useTheme';
import { ACCENTS } from '@/lib/theme/accents';
import type { GroupSummary } from '@/schemas/client';
import { SizeFormatter } from '@/utils';

// A group has no stored colour, so its name picks one; the same group keeps the same colour.
export function groupAccent(name: string): string {
  let h = 0;
  for (const ch of name) h = (h * 31 + ch.codePointAt(0)!) >>> 0;
  return ACCENTS[h % ACCENTS.length].id;
}

export default function GroupCard({
  group: g,
  share,
  onRename,
  actions,
}: {
  group: GroupSummary;
  share: number;
  onRename: () => void;
  actions: MenuAction[];
}) {
  const { t } = useTranslation();
  const num = useNum();
  const { isDark } = useTheme();
  const scope = accentScope(groupAccent(g.name), '', isDark);
  return (
    <section className={`tile edge s6 ${scope.className}`} style={scope.style} aria-label={g.name}>
      <div className="row sp" style={{ marginBottom: 12 }}>
        <div className="row" style={{ minWidth: 0 }}>
          <span className="ico acc">
            <Icon name="groups" />
          </span>
          <div className="h3" style={{ overflowWrap: 'anywhere' }}>
            {g.name}
          </div>
        </div>
        <div className="row" style={{ gap: 4 }}>
          <IconBtn
            small
            icon="edit"
            label={`${t('pages.groups.rename')} ${g.name}`}
            onClick={onRename}
          />
          <MoreMenu label={`${t('pages.clients.actions')} ${g.name}`} actions={actions} />
        </div>
      </div>
      <div className="row wrap" style={{ gap: 24 }}>
        <div>
          <div className="kpi num" style={{ fontSize: 30 }}>
            {num.int(g.clientCount)}
          </div>
          <div className="lbl">{t('groups.members')}</div>
        </div>
        <div className="f1" style={{ minWidth: 200 }}>
          <Meter
            label={t('groups.share')}
            value={share}
            note={
              <bdi>{`${SizeFormatter.sizeFormat(g.trafficUsed || 0)} · ↑ ${SizeFormatter.sizeFormat(g.up || 0)} · ↓ ${SizeFormatter.sizeFormat(g.down || 0)}`}</bdi>
            }
          />
        </div>
      </div>
    </section>
  );
}
