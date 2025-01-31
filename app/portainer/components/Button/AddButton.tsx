import clsx from 'clsx';

import styles from './AddButton.module.css';

export interface Props {
  label: string;
  onClick: () => void;
}

export function AddButton({ label, onClick }: Props) {
  return (
    <button
      className={clsx(
        'label',
        'label-default',
        'interactive',
        styles.addButton
      )}
      type="button"
      onClick={onClick}
    >
      <i className="fa fa-plus-circle space-right" aria-hidden="true" /> {label}
    </button>
  );
}
