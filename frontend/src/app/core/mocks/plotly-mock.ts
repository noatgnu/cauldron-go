import { vi } from 'vitest';

export const PlotlyMock = {
  version: '2.35.2',
  newPlot: (el: any) => {
    const instance = el as any;
    instance.on = () => {};
    instance.removeListener = () => {};
    instance.removeAllListeners = () => {};
    return Promise.resolve(instance);
  },
  Plots: {
    resize: () => {}
  },
  react: (el: any) => {
    const instance = el as any;
    instance.on = () => {};
    instance.removeListener = () => {};
    instance.removeAllListeners = () => {};
    return Promise.resolve(instance);
  },
  purge: () => {}
};

export const mockMatchMedia = () => {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation(query => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
};
