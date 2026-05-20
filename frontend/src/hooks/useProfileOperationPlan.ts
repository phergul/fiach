import { useCallback, useEffect, useState } from 'react';

import { BuildProfileOperationPlan } from '@bindings/github.com/phergul/mod-manager/internal/services/profileservice';
import type { OperationPlan } from '@bindings/github.com/phergul/mod-manager/internal/operationplan/models';
import { getErrorMessage } from '@utils';

export const useProfileOperationPlan = (profileID: number | null) => {
  const [plan, setPlan] = useState<OperationPlan | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const loadPlan = useCallback(async () => {
    if (profileID === null) {
      setPlan(null);
      setIsLoading(false);
      setLoadError(null);
      return null;
    }

    setIsLoading(true);
    setLoadError(null);

    try {
      const loadedPlan = await BuildProfileOperationPlan(profileID);
      setPlan(loadedPlan);
      return loadedPlan;
    } catch (error) {
      const message = getErrorMessage(error);
      setLoadError(message);
      throw error;
    } finally {
      setIsLoading(false);
    }
  }, [profileID]);

  useEffect(() => {
    let isMounted = true;

    const loadInitialPlan = async () => {
      if (profileID === null) {
        setPlan(null);
        setIsLoading(false);
        setLoadError(null);
        return;
      }

      setIsLoading(true);
      setLoadError(null);

      try {
        const loadedPlan = await BuildProfileOperationPlan(profileID);
        if (isMounted) {
          setPlan(loadedPlan);
        }
      } catch (error) {
        if (isMounted) {
          setLoadError(getErrorMessage(error));
        }
      } finally {
        if (isMounted) {
          setIsLoading(false);
        }
      }
    };

    loadInitialPlan();

    return () => {
      isMounted = false;
    };
  }, [profileID]);

  return {
    isLoading,
    loadError,
    plan,
    refreshPlan: loadPlan,
  };
};

export type UseProfileOperationPlanResult = ReturnType<typeof useProfileOperationPlan>;
