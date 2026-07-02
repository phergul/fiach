import { useCallback, useEffect, useState } from 'react';

import { Window } from '@wailsio/runtime';

export const useWindowMaximised = () => {
  const [isMaximised, setIsMaximised] = useState(false);

  useEffect(() => {
    let isActive = true;

    void Window.IsMaximised().then((maximised) => {
      if (isActive) {
        setIsMaximised(maximised);
      }
    });

    return () => {
      isActive = false;
    };
  }, []);

  const toggleMaximised = useCallback(async () => {
    const maximised = await Window.IsMaximised();
    if (maximised) {
      await Window.Restore();
    } else {
      await Window.Maximise();
    }
    setIsMaximised(!maximised);
  }, []);

  return { isMaximised, toggleMaximised };
};
