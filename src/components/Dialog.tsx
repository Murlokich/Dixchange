import { useEffect, type ReactNode } from 'react';

type DialogProps = {
  title: string;
  titleId: string;
  className?: string;
  // While busy (e.g. a request in flight) the dialog cannot be dismissed.
  isBusy?: boolean;
  onClose: () => void;
  // When given, the dialog is a form submitted by its submit button; otherwise a plain window.
  onSubmit?: () => void;
  children: ReactNode;
};

// Modal shell: closes on Escape, backdrop click or the caller's Cancel/Close button.
export default function Dialog({
  title,
  titleId,
  className,
  isBusy = false,
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

  const dialogProps = {
    className: `dialog ${className ?? ''}`,
    role: 'dialog',
    'aria-modal': true,
    'aria-labelledby': titleId,
  };
  const content = (
    <>
      <h2 id={titleId}>{title}</h2>
      {children}
    </>
  );

  return (
    <div
      className="dialog-backdrop"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget && !isBusy) {
          onClose();
        }
      }}
    >
      {onSubmit ? (
        <form
          {...dialogProps}
          onSubmit={(event) => {
            event.preventDefault();
            onSubmit();
          }}
        >
          {content}
        </form>
      ) : (
        <div {...dialogProps}>{content}</div>
      )}
    </div>
  );
}
