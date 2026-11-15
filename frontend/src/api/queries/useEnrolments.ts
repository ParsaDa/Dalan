import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { HttpUtil } from '@/utils';
import { parseMsg } from '@/utils/zodValidate';
import {
  InvitationSchema,
  NodeEnrolmentSchema,
  type Invitation,
  type NodeEnrolment,
} from '@/generated/zod';
import type { EnrolmentFormValues } from '@/schemas/enrolment';

export type { Invitation, NodeEnrolment };

const JSON_HEADERS = { headers: { 'Content-Type': 'application/json' } };
const ENROLMENTS_KEY = ['enrolments'] as const;

async function fetchEnrolments(): Promise<NodeEnrolment[]> {
  const msg = await HttpUtil.get('/panel/api/enrolments/list', undefined, { silent: true });
  if (!msg?.success) throw new Error(msg?.msg || 'Failed to fetch enrolments');
  return parseMsg(msg, z.array(NodeEnrolmentSchema), 'enrolments/list').obj ?? [];
}

// Rows with a node id belong to joined nodes and are shown in the node list instead.
export function useWaitingEnrolments(enabled: boolean) {
  const query = useQuery({ queryKey: ENROLMENTS_KEY, queryFn: fetchEnrolments, enabled });
  return (query.data ?? []).filter((row) => row.nodeId === 0);
}

export function useEnrolmentMutations() {
  const queryClient = useQueryClient();
  const refresh = () => queryClient.invalidateQueries({ queryKey: ENROLMENTS_KEY });

  const create = useMutation({
    mutationFn: async (values: EnrolmentFormValues): Promise<Invitation | null> => {
      const msg = await HttpUtil.post('/panel/api/enrolments/add', values, JSON_HEADERS);
      if (!msg?.success) return null;
      return parseMsg(msg, InvitationSchema, 'enrolments/add').obj ?? null;
    },
    onSuccess: refresh,
  });
  const cancel = useMutation({
    mutationFn: (id: number) => HttpUtil.post(`/panel/api/enrolments/del/${id}`),
    onSuccess: refresh,
  });
  return { create: create.mutateAsync, creating: create.isPending, cancel: cancel.mutateAsync };
}
