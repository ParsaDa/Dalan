import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Alert, Button } from 'antd';

import { useNodesToEnrolAgain } from '@/api/queries/useEnrolments';
import type { NodeRecord } from '@/api/queries/useNodesQuery';
import { EnrolNodeDialog } from './EnrolNode';

// A node that joined through the removed Xray tunnel has lost its link until it is enrolled again.
export default function ReenrolNotices({ nodes }: { nodes: NodeRecord[] }) {
  const { t } = useTranslation();
  const [name, setName] = useState<string>();
  const rows = useNodesToEnrolAgain().filter((row) => nodes.some((node) => node.id === row.nodeId));

  return (
    <>
      {rows.map((row) => (
        <Alert
          key={row.id}
          type="warning"
          showIcon
          style={{ marginBottom: 8 }}
          title={t('pages.enrolments.needsReenrol', { name: row.name })}
          description={t('pages.enrolments.needsReenrolHint')}
          action={
            <Button size="small" onClick={() => setName(row.name)}>
              {t('pages.enrolments.reenrol')}
            </Button>
          }
        />
      ))}
      <EnrolNodeDialog
        open={name !== undefined}
        initialName={name}
        onClose={() => setName(undefined)}
      />
    </>
  );
}
