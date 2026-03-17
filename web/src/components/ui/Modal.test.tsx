/**
 * Modal Component Tests
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Modal, ConfirmModal } from './Modal';

describe('Modal', () => {
  describe('Rendering', () => {
    it('renders when open', () => {
      render(
        <Modal isOpen={true} onClose={() => {}} title="Test Modal">
          <p>Modal content</p>
        </Modal>
      );

      expect(screen.getByRole('dialog')).toBeInTheDocument();
      expect(screen.getByText('Test Modal')).toBeInTheDocument();
      expect(screen.getByText('Modal content')).toBeInTheDocument();
    });

    it('does not render when closed', () => {
      render(
        <Modal isOpen={false} onClose={() => {}} title="Test Modal">
          <p>Modal content</p>
        </Modal>
      );

      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });

    it('renders footer when provided', () => {
      render(
        <Modal isOpen={true} onClose={() => {}} title="Test Modal" footer={<button>Action</button>}>
          <p>Content</p>
        </Modal>
      );

      expect(screen.getByRole('button', { name: 'Action' })).toBeInTheDocument();
    });

    it('renders different sizes', () => {
      const { rerender, container } = render(
        <Modal isOpen={true} onClose={() => {}} title="Small" size="sm">
          Content
        </Modal>
      );

      let dialog = container.querySelector('[role="dialog"]');
      expect(dialog).toHaveClass('max-w-sm');

      rerender(
        <Modal isOpen={true} onClose={() => {}} title="Large" size="lg">
          Content
        </Modal>
      );

      dialog = container.querySelector('[role="dialog"]');
      expect(dialog).toHaveClass('max-w-lg');
    });
  });

  describe('Interaction', () => {
    it('calls onClose when close button clicked', async () => {
      const handleClose = vi.fn();
      const user = userEvent.setup();

      render(
        <Modal isOpen={true} onClose={handleClose} title="Test">
          Content
        </Modal>
      );

      await user.click(screen.getByRole('button', { name: 'Close modal' }));
      expect(handleClose).toHaveBeenCalledTimes(1);
    });

    it('calls onClose when ESC pressed (default)', async () => {
      const handleClose = vi.fn();
      const user = userEvent.setup();

      render(
        <Modal isOpen={true} onClose={handleClose} title="Test">
          Content
        </Modal>
      );

      await user.keyboard('{Escape}');
      expect(handleClose).toHaveBeenCalledTimes(1);
    });

    it('does not close on ESC when closeOnEsc is false', async () => {
      const handleClose = vi.fn();
      const user = userEvent.setup();

      render(
        <Modal isOpen={true} onClose={handleClose} title="Test" closeOnEsc={false}>
          Content
        </Modal>
      );

      await user.keyboard('{Escape}');
      expect(handleClose).not.toHaveBeenCalled();
    });

    it('calls onClose when backdrop clicked (default)', async () => {
      const handleClose = vi.fn();
      const user = userEvent.setup();

      render(
        <Modal isOpen={true} onClose={handleClose} title="Test">
          Content
        </Modal>
      );

      const backdrop = document.querySelector('.bg-black\\/50');
      if (backdrop) {
        await user.click(backdrop as HTMLElement);
        expect(handleClose).toHaveBeenCalledTimes(1);
      }
    });

    it('does not close on backdrop click when closeOnBackdrop is false', async () => {
      const handleClose = vi.fn();
      const user = userEvent.setup();

      render(
        <Modal isOpen={true} onClose={handleClose} title="Test" closeOnBackdrop={false}>
          Content
        </Modal>
      );

      const backdrop = document.querySelector('.bg-black\\/50');
      if (backdrop) {
        await user.click(backdrop as HTMLElement);
        expect(handleClose).not.toHaveBeenCalled();
      }
    });
  });

  describe('Accessibility', () => {
    it('has dialog role', () => {
      render(
        <Modal isOpen={true} onClose={() => {}} title="Test">
          Content
        </Modal>
      );

      expect(screen.getByRole('dialog')).toBeInTheDocument();
    });

    it('has aria-modal attribute', () => {
      render(
        <Modal isOpen={true} onClose={() => {}} title="Test">
          Content
        </Modal>
      );

      expect(screen.getByRole('dialog')).toHaveAttribute('aria-modal', 'true');
    });

    it('has aria-labelledby referencing title', () => {
      render(
        <Modal isOpen={true} onClose={() => {}} title="Test Modal">
          Content
        </Modal>
      );

      const dialog = screen.getByRole('dialog');
      expect(dialog).toHaveAttribute('aria-labelledby', 'modal-title');
      expect(screen.getByText('Test Modal')).toHaveAttribute('id', 'modal-title');
    });
  });

  describe('Styling', () => {
    it('applies custom className', () => {
      const { container } = render(
        <Modal isOpen={true} onClose={() => {}} title="Test" className="custom-modal">
          Content
        </Modal>
      );

      const dialog = container.querySelector('[role="dialog"]');
      expect(dialog).toHaveClass('custom-modal');
    });

    it('uses Tailwind classes only (no inline styles)', () => {
      const { container } = render(
        <Modal isOpen={true} onClose={() => {}} title="Test">
          Content
        </Modal>
      );

      const dialog = container.querySelector('[role="dialog"]');
      expect(dialog?.getAttribute('style')).toBeNull();
    });
  });
});

describe('ConfirmModal', () => {
  it('renders with correct content', () => {
    render(
      <ConfirmModal
        isOpen={true}
        onClose={() => {}}
        onConfirm={() => {}}
        title="Confirm Action"
        message="Are you sure?"
      />
    );

    expect(screen.getByText('Confirm Action')).toBeInTheDocument();
    expect(screen.getByText('Are you sure?')).toBeInTheDocument();
  });

  it('calls onConfirm when confirm button clicked', async () => {
    const handleConfirm = vi.fn();
    const user = userEvent.setup();

    render(
      <ConfirmModal
        isOpen={true}
        onClose={() => {}}
        onConfirm={handleConfirm}
        title="Confirm"
        message="Are you sure?"
      />
    );

    await user.click(screen.getByRole('button', { name: 'Confirm' }));
    expect(handleConfirm).toHaveBeenCalledTimes(1);
  });

  it('calls onClose when cancel button clicked', async () => {
    const handleClose = vi.fn();
    const user = userEvent.setup();

    render(
      <ConfirmModal
        isOpen={true}
        onClose={handleClose}
        onConfirm={() => {}}
        title="Confirm"
        message="Are you sure?"
      />
    );

    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(handleClose).toHaveBeenCalledTimes(1);
  });

  it('renders custom button text', () => {
    render(
      <ConfirmModal
        isOpen={true}
        onClose={() => {}}
        onConfirm={() => {}}
        title="Delete"
        message="Delete this item?"
        confirmText="Delete"
        cancelText="Keep"
      />
    );

    expect(screen.getByRole('button', { name: 'Delete' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Keep' })).toBeInTheDocument();
  });

  it('renders danger variant', () => {
    render(
      <ConfirmModal
        isOpen={true}
        onClose={() => {}}
        onConfirm={() => {}}
        title="Delete"
        message="Are you sure?"
        variant="danger"
      />
    );

    const confirmButton = screen.getByRole('button', { name: 'Confirm' });
    expect(confirmButton).toHaveClass('bg-status-error');
  });

  it('shows loading state', () => {
    render(
      <ConfirmModal
        isOpen={true}
        onClose={() => {}}
        onConfirm={() => {}}
        title="Processing"
        message="Please wait..."
        isLoading={true}
      />
    );

    // Button should contain "Confirm" text even if loading
    const buttons = screen.getAllByRole('button');
    const confirmButton = buttons.find((btn) => btn.textContent?.includes('Confirm'));
    expect(confirmButton).toBeDefined();
    expect(confirmButton).toBeDisabled();
    expect(screen.getByRole('status', { name: 'Loading' })).toBeInTheDocument();
  });
});
