import { useTranslation } from 'react-i18next';

import { Modal } from '@/components/dl/overlays';
import TorPanel from '@/pages/tor/TorPanel';

interface TorModalProps {
  open: boolean;
  onClose: () => void;
  onChanged?: () => void;
}

export default function TorModal({ open, onClose, onChanged }: TorModalProps) {
  const { t } = useTranslation();
  return (
    <Modal open={open} onClose={onClose} label={t('pages.tor.title')} width={680}>
      <h2 className="h2">{t('pages.tor.title')}</h2>
      {open && <TorPanel onChanged={onChanged} />}
    </Modal>
  );
}
