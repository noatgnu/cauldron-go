import { describe, it, expect, afterEach } from 'vitest';
import { pickBrowserFile } from './browser-file-picker';

function findFileInput(): HTMLInputElement {
  const inputs = document.querySelectorAll('input[type="file"]');
  const input = inputs[inputs.length - 1];
  if (!input) throw new Error('expected a file input to be appended to the document');
  return input as HTMLInputElement;
}

afterEach(() => {
  document.querySelectorAll('input[type="file"]').forEach((el) => el.remove());
});

function selectFile(input: HTMLInputElement, file: File): void {
  Object.defineProperty(input, 'files', { value: [file], configurable: true });
  input.dispatchEvent(new Event('change'));
}

describe('pickBrowserFile', () => {
  it('appends a hidden file input to the document and triggers it', () => {
    const promise = pickBrowserFile();
    const input = findFileInput();
    expect(input.style.display).toBe('none');
    input.dispatchEvent(new Event('cancel'));
    return promise;
  });

  it('applies the accept filter when given', () => {
    const promise = pickBrowserFile('.tsv,.csv');
    const input = findFileInput();
    expect(input.accept).toBe('.tsv,.csv');
    input.dispatchEvent(new Event('cancel'));
    return promise;
  });

  it('resolves with the selected file', async () => {
    const promise = pickBrowserFile();
    const input = findFileInput();
    const file = new File(['a\tb'], 'sample.tsv');

    selectFile(input, file);

    const result = await promise;
    expect(result).toBe(file);
  });

  it('removes the input from the document after selection', async () => {
    const promise = pickBrowserFile();
    const input = findFileInput();
    selectFile(input, new File(['x'], 'x.txt'));
    await promise;

    expect(document.body.contains(input)).toBe(false);
  });

  it('resolves with null when the user cancels', async () => {
    const promise = pickBrowserFile();
    const input = findFileInput();

    input.dispatchEvent(new Event('cancel'));

    const result = await promise;
    expect(result).toBeNull();
  });

  it('ignores a cancel event that arrives after a change event', async () => {
    const promise = pickBrowserFile();
    const input = findFileInput();
    const file = new File(['x'], 'x.txt');

    selectFile(input, file);
    input.dispatchEvent(new Event('cancel'));

    const result = await promise;
    expect(result).toBe(file);
  });
});
