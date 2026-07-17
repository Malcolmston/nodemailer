import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { DocsView } from '../../../src/components/DocsView';
import type { DocIndex } from 'go-ui';

// A minimal DocIndex the stubbed fetch returns for DocsApp's doc.json request.
const DOC_INDEX: DocIndex = {
  module: 'github.com/malcolmston/nodemailer',
  packages: [
    {
      importPath: 'github.com/malcolmston/nodemailer',
      name: 'nodemailer',
      synopsis: 'Package nodemailer is a standard-library-only Nodemailer-style email library.',
      doc: 'Package nodemailer is a standard-library-only Nodemailer-style email library.',
      consts: [],
      vars: [],
      types: [
        {
          name: 'Message',
          signature: 'type Message struct{}',
          doc: 'Message is a builder for an RFC 5322 / MIME email message.',
          consts: [],
          vars: [],
          funcs: [],
          methods: [],
        },
      ],
      funcs: [{ name: 'New', signature: 'func New() *Message', doc: 'New returns an empty Message.' }],
    },
  ],
};

describe('DocsView', () => {
  beforeEach(() => {
    // DocsApp fetches doc.json; return the small index.
    global.fetch = vi.fn((input: RequestInfo | URL) => {
      if (String(input).includes('doc.json')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(DOC_INDEX) } as Response);
      }
      return new Promise<Response>(() => {});
    }) as unknown as typeof fetch;
  });

  it('renders the inline React API reference from the fetched doc.json', async () => {
    const { container } = render(<DocsView />);
    expect(container.querySelector('#view-docs')).not.toBeNull();
    expect(
      screen.getByRole('heading', { level: 2, name: /API documentation/ }),
    ).toBeInTheDocument();

    // DocsApp fetches asynchronously, then renders the package view + symbols.
    expect(await screen.findByRole('heading', { name: /package nodemailer/ })).toBeInTheDocument();
    expect(container.querySelector('#sym-New'), 'func New symbol card').not.toBeNull();
    expect(container.querySelector('#sym-Message'), 'type Message symbol card').not.toBeNull();

    // The secondary link to the raw generated static HTML remains.
    expect(screen.getByRole('link', { name: /Open the raw generated HTML/ })).toHaveAttribute('href', './api/');
  });
});
