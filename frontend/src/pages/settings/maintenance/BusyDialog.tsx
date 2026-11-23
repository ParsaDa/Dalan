import Mark from '@/components/dl/Mark';
import { Modal as DlModal } from '@/components/dl/overlays';

export interface BusyEvent {
  busy: boolean;
  tip?: string;
}

// Blocks the page while a restore, update or Xray switch restarts things.
export function BusyDialog({ busy, tip }: { busy: boolean; tip: string }) {
  return (
    <DlModal open={busy} onClose={() => {}} label={tip} width={420}>
      <div className="row" aria-live="polite">
        <Mark size={40} motion="loop" />
        <b>{tip}</b>
      </div>
    </DlModal>
  );
}
