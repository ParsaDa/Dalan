import { useTranslation } from 'react-i18next';

import Flag from '@/components/ui/Flag';
import type { NodeRecord } from '@/api/queries/useNodesQuery';

interface NodeTabsProps {
  nodes: NodeRecord[];
  activeId: number | null;
  onSelect: (id: number | null) => void;
}

function dotClass(node: NodeRecord): string {
  if (node.status === 'online') return 'ok';
  if (node.status === 'offline') return 'bad';
  return 'idle';
}

export default function NodeTabs({ nodes, activeId, onSelect }: NodeTabsProps) {
  const { t } = useTranslation();
  return (
    <div className="ntabs" role="tablist">
      <button
        type="button"
        role="tab"
        aria-selected={activeId === null}
        className={`ntab${activeId === null ? ' on' : ''}`}
        onClick={() => onSelect(null)}
      >
        {t('pages.index.thisServer')}
      </button>
      {nodes
        .filter((node) => node.enable !== false)
        .map((node) => (
          <button
            key={node.id}
            type="button"
            role="tab"
            aria-selected={activeId === node.id}
            className={`ntab${activeId === node.id ? ' on' : ''}`}
            onClick={() => onSelect(node.id)}
          >
            {node.country && (
              <span aria-hidden>
                <Flag country={node.country} size={16} />
              </span>
            )}
            <span>{node.name}</span>
            <i className={`ntab-dot ${dotClass(node)}`} aria-hidden />
          </button>
        ))}
    </div>
  );
}
