/**
 * Prompts the user to pick one file via a hidden native browser file input.
 * Used in server mode, where there is no shared filesystem to show a native OS dialog for.
 */
export function pickBrowserFile(accept?: string): Promise<File | null> {
  return new Promise((resolve) => {
    const input = document.createElement('input');
    input.type = 'file';
    if (accept) {
      input.accept = accept;
    }
    input.style.display = 'none';

    let settled = false;
    const settle = (file: File | null) => {
      if (settled) return;
      settled = true;
      input.remove();
      resolve(file);
    };

    input.addEventListener('change', () => {
      settle(input.files?.[0] ?? null);
    });
    input.addEventListener('cancel', () => {
      settle(null);
    });

    document.body.appendChild(input);
    input.click();
  });
}
