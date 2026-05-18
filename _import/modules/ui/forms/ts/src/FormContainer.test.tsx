// @vitest-environment jsdom

import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { FormContainer } from './FormContainer';

describe('FormContainer', () => {
  it('wraps children in a form, forwards submit, and preserves custom classes', () => {
    const onSubmit = vi.fn((event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
    });

    render(
      <FormContainer onSubmit={onSubmit} className="custom-shell">
        <button type="submit">Save</button>
      </FormContainer>,
    );

    const button = screen.getByRole('button', { name: 'Save' });
    const form = button.closest('form');
    const wrapper = form?.parentElement;

    fireEvent.submit(form as HTMLFormElement);

    expect(onSubmit).toHaveBeenCalledTimes(1);
    expect(wrapper?.className.includes('custom-shell')).toBe(true);
  });
});
