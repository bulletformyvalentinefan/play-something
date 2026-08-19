import Modal from './Modal'

export default function ConfirmModal({ title, message, confirmLabel = 'eliminar', busy, onConfirm, onClose }) {
  return (
    <Modal title={title} onClose={onClose}>
      <p className="muted">{message}</p>
      <div className="segment">
        <button type="button" className="primary-btn" onClick={onConfirm} disabled={busy}>
          {busy ? 'eliminando…' : confirmLabel}
        </button>
        <button type="button" className="theme-btn" onClick={onClose} disabled={busy}>
          cancelar
        </button>
      </div>
    </Modal>
  )
}