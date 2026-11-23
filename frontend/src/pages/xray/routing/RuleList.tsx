import type { PointerEvent } from 'react';
import { useTranslation } from 'react-i18next';

import Icon from '@/components/dl/Icon';
import { MoreMenu } from '@/components/dl/MoreMenu';
import { Badge, IconBtn, Toggle } from '@/components/dl/primitives';
import { useNum } from '@/components/dl/numbers';
import { isApiRule, ruleCriteriaChips } from './helpers';
import type { RuleRow } from './types';

interface RuleListProps {
  rows: RuleRow[];
  draggedIndex: number | null;
  dropTargetIndex: number | null;
  onHandlePointerDown: (idx: number, ev: PointerEvent) => void;
  openEdit: (idx: number) => void;
  moveUp: (idx: number) => void;
  moveDown: (idx: number) => void;
  confirmDelete: (idx: number) => void;
  toggleRule: (idx: number, enabled: boolean) => void;
}

// What a rule matches, in one line: its criteria, else its inbounds, else everything.
export function ruleMatch(rule: RuleRow): string {
  const chips = ruleCriteriaChips(rule).map((c) => `${c.label.toLowerCase()} ${c.value}`);
  if (chips.length) return chips.join(' · ');
  if (rule.inboundTag) return `inbound ${rule.inboundTag}`;
  return 'any';
}

// The design's rule list: checked top to bottom, dragged by the handle to reorder.
export default function RuleList(p: RuleListProps) {
  const { t } = useTranslation();
  const num = useNum();
  return (
    <div className="list">
      {p.rows.map((rule, i) => {
        const target = rule.outboundTag || rule.balancerTag || '';
        const dragging = p.draggedIndex === i;
        const dropHere = p.draggedIndex != null && p.dropTargetIndex === i && !dragging;
        const title = rule.comment || ruleMatch(rule);
        return (
          <div
            key={rule.key}
            className="li rule-row"
            data-row-key={i}
            style={{
              flexWrap: 'wrap',
              opacity: rule.enabled === false ? 0.55 : dragging ? 0.6 : 1,
              boxShadow: dropHere ? '0 0 0 2px var(--acc-line)' : undefined,
            }}
          >
            <span className="badge num">{num.int(i + 1)}</span>
            <button
              type="button"
              className="ib sm drag-handle"
              aria-label={t('routing.drag')}
              style={{ cursor: 'grab', touchAction: 'none' }}
              onPointerDown={(ev) => p.onHandlePointerDown(i, ev)}
            >
              <Icon name="menu" />
            </button>
            <div style={{ flex: '1 1 160px', minWidth: 0 }}>
              <div style={{ fontWeight: 750, overflowWrap: 'anywhere' }}>{title}</div>
              {rule.comment && (
                <div className="mut mono" style={{ fontSize: 12, overflowWrap: 'anywhere' }}>
                  <bdi>{ruleMatch(rule)}</bdi>
                </div>
              )}
            </div>
            <div className="row" style={{ gap: 6, marginInlineStart: 'auto' }}>
              <span className="mut">
                <Icon name="chev" size={16} className="flipx" />
              </span>
              {target && (
                <Badge acc>{rule.balancerTag && !rule.outboundTag ? `⚖ ${target}` : target}</Badge>
              )}
              <Toggle
                on={rule.enabled !== false}
                label={`${t('routing.enabled')} ${title}`}
                disabled={isApiRule(rule)}
                onChange={(on) => p.toggleRule(i, on)}
              />
              <IconBtn small icon="edit" label={t('edit')} onClick={() => p.openEdit(i)} />
              <MoreMenu
                label={`${t('pages.nodes.actions')} ${title}`}
                actions={[
                  {
                    key: 'up',
                    label: t('pages.inbounds.form.moveUp'),
                    disabled: i === 0,
                    onSelect: () => p.moveUp(i),
                  },
                  {
                    key: 'down',
                    label: t('pages.inbounds.form.moveDown'),
                    disabled: i === p.rows.length - 1,
                    onSelect: () => p.moveDown(i),
                  },
                  {
                    key: 'del',
                    label: t('delete'),
                    icon: 'trash',
                    danger: true,
                    onSelect: () => p.confirmDelete(i),
                  },
                ]}
              />
            </div>
          </div>
        );
      })}
    </div>
  );
}
