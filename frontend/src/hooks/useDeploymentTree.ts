import { useCallback, useEffect, useRef, useState } from 'react';

import { LoadDeploymentTreeChildren } from '@bindings/github.com/phergul/fiach/internal/services/deploymentreviewservice';
import type { DeploymentTreeNode } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import {
  collectDirectoryPaths,
  hasActiveDeploymentTreeFilters,
  type DeploymentTreeFilters,
} from '@utils';
import { getErrorMessage } from '@utils';

const ROOT_PATH = '';
export const DEPLOYMENT_TREE_SCAN_CAP = 500;

const buildInitialChildrenMap = (rootChildren: DeploymentTreeNode[]) => ({
  [ROOT_PATH]: rootChildren,
});

export const useDeploymentTree = (
  previewHash: string,
  rootChildren: DeploymentTreeNode[],
  filters: DeploymentTreeFilters,
) => {
  const [childrenByParent, setChildrenByParent] = useState<Record<string, DeploymentTreeNode[]>>(
    () => buildInitialChildrenMap(rootChildren),
  );
  const [expandedPaths, setExpandedPaths] = useState<Record<string, boolean>>({});
  const [loadingPaths, setLoadingPaths] = useState<Record<string, boolean>>({});
  const [loadErrors, setLoadErrors] = useState<Record<string, string>>({});
  const [isScanning, setIsScanning] = useState(false);
  const [scanCapReached, setScanCapReached] = useState(false);
  const inFlightLoads = useRef<Set<string>>(new Set());
  const loadedPaths = useRef<Set<string>>(new Set([ROOT_PATH]));
  const childrenByParentRef = useRef(childrenByParent);

  childrenByParentRef.current = childrenByParent;

  useEffect(() => {
    inFlightLoads.current = new Set();
    loadedPaths.current = new Set([ROOT_PATH]);
    setChildrenByParent(buildInitialChildrenMap(rootChildren));
    setExpandedPaths({});
    setLoadingPaths({});
    setLoadErrors({});
    setIsScanning(false);
    setScanCapReached(false);
  }, [previewHash, rootChildren]);

  const ensureChildrenLoaded = useCallback(
    async (parentPath: string) => {
      if (previewHash === '') {
        return [];
      }

      if (loadedPaths.current.has(parentPath)) {
        return childrenByParentRef.current[parentPath] ?? [];
      }

      if (inFlightLoads.current.has(parentPath)) {
        return childrenByParentRef.current[parentPath] ?? [];
      }

      inFlightLoads.current.add(parentPath);
      setLoadingPaths((currentLoadingPaths) => ({
        ...currentLoadingPaths,
        [parentPath]: true,
      }));
      setLoadErrors((currentLoadErrors) => {
        const nextLoadErrors = { ...currentLoadErrors };
        delete nextLoadErrors[parentPath];
        return nextLoadErrors;
      });

      try {
        const children = await LoadDeploymentTreeChildren(previewHash, parentPath);
        loadedPaths.current.add(parentPath);
        setChildrenByParent((currentChildrenByParent) => ({
          ...currentChildrenByParent,
          [parentPath]: children,
        }));
        return children;
      } catch (error) {
        setLoadErrors((currentLoadErrors) => ({
          ...currentLoadErrors,
          [parentPath]: getErrorMessage(error),
        }));
        return [];
      } finally {
        inFlightLoads.current.delete(parentPath);
        setLoadingPaths((currentLoadingPaths) => {
          const nextLoadingPaths = { ...currentLoadingPaths };
          delete nextLoadingPaths[parentPath];
          return nextLoadingPaths;
        });
      }
    },
    [previewHash],
  );

  const expandNode = useCallback(
    async (path: string) => {
      setExpandedPaths((currentExpandedPaths) => ({
        ...currentExpandedPaths,
        [path]: true,
      }));
      await ensureChildrenLoaded(path);
    },
    [ensureChildrenLoaded],
  );

  const collapseNode = useCallback((path: string) => {
    setExpandedPaths((currentExpandedPaths) => {
      const nextExpandedPaths = { ...currentExpandedPaths };
      delete nextExpandedPaths[path];
      return nextExpandedPaths;
    });
  }, []);

  const toggleNode = useCallback(
    async (path: string, isExpanded: boolean) => {
      if (isExpanded) {
        collapseNode(path);
        return;
      }

      await expandNode(path);
    },
    [collapseNode, expandNode],
  );

  const filterKey = `${filters.statuses.join('|')}::${filters.risks.join('|')}::${filters.searchQuery}`;

  useEffect(() => {
    if (!hasActiveDeploymentTreeFilters(filters) || previewHash === '') {
      setIsScanning(false);
      setScanCapReached(false);
      return;
    }

    let isCancelled = false;

    const scanTree = async () => {
      setIsScanning(true);
      setScanCapReached(false);

      const queue = rootChildren
        .filter((node) => node.IsDirectory && node.HasChildren)
        .map((node) => node.Path);
      let processedCount = 0;

      while (queue.length > 0 && !isCancelled) {
        const parentPath = queue.shift();
        if (parentPath === undefined) {
          continue;
        }

        const children = await ensureChildrenLoaded(parentPath);
        processedCount += 1;

        if (processedCount >= DEPLOYMENT_TREE_SCAN_CAP) {
          if (!isCancelled) {
            setScanCapReached(queue.length > 0);
          }
          break;
        }

        for (const child of children) {
          if (child.IsDirectory && child.HasChildren) {
            queue.push(child.Path);
          }
        }
      }

      if (!isCancelled) {
        setIsScanning(false);
      }
    };

    scanTree();

    return () => {
      isCancelled = true;
    };
  }, [ensureChildrenLoaded, filterKey, filters, previewHash, rootChildren]);

  const getChildren = useCallback(
    (parentPath: string) => {
      return childrenByParent[parentPath] ?? [];
    },
    [childrenByParent],
  );

  const loadedDirectoryCount = collectDirectoryPaths(
    childrenByParent[ROOT_PATH] ?? rootChildren,
    childrenByParent,
  ).filter((path) => loadedPaths.current.has(path)).length;

  return {
    collapseNode,
    ensureChildrenLoaded,
    expandNode,
    expandedPaths,
    getChildren,
    isScanning,
    loadErrors,
    loadedDirectoryCount,
    loadingPaths,
    scanCapReached,
    toggleNode,
  };
};

export type UseDeploymentTreeResult = ReturnType<typeof useDeploymentTree>;
