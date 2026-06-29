import { useEffect, useState } from 'react';
import { OS } from '@bindings/github.com/phergul/fiach/internal/appmode/runtime';

export function useRuntime() {
  const [runtime, setRuntime] = useState<string>('unknown');

  useEffect(() => {
    void OS().then(setRuntime);
  }, []);

  return {
    runtime: runtime ?? 'unknown',
    isWindows: runtime === 'windows',
    isMac: runtime === 'darwin',
    isLinux: runtime === 'linux',
  };
}
