import { useEffect, type ReactNode } from 'react';

type DialogProps = {
  title: string;
  titleId: string;
  className?: string;
  // While busy (e.g. a request in flight) the dialog cannot be dismissed.
  isBusy: boolean;
  onClose: () => void;
  onSubmit: () => void;
  children: ReactNode;
};

// Modal form shell: closes on Escape, backdrop click or the caller's Cancel button.
export default function Dialog({
  title,
  titleId,
  className,
  isBusy,
  onClose,
  onSubmit,
  children,
}: DialogProps) {
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !isBusy) {
        onClose();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isBusy, onClose]);

  return (
    <div
      className="dialog-backdrop"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget && !isBusy) {
          onClose();
        }
      }}
    >
      <form
        className={`dialog ${className ?? ''}`}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        onSubmit={(event) => {
          event.preventDefault();
          onSubmit();
        }}
      >
        <h2 id={titleId}>{title}</h2>
        {children}
      </form>
    </div>
  );
}
