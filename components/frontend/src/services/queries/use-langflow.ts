/**
 * React Query hooks for LangFlow operations
 */

import { useQuery } from '@tanstack/react-query';
import { langflowApi } from '../api/langflow';

/**
 * Query key factory for LangFlow queries
 */
export const langflowKeys = {
  all: ['langflow'] as const,
  health: () => [...langflowKeys.all, 'health'] as const,
  flows: () => [...langflowKeys.all, 'flows'] as const,
  flow: (id: string) => [...langflowKeys.all, 'flow', id] as const,
};

/**
 * Hook to check LangFlow service health
 * Refetches every 30 seconds to keep health status current
 */
export function useLangFlowHealth() {
  return useQuery({
    queryKey: langflowKeys.health(),
    queryFn: langflowApi.checkHealth,
    refetchInterval: 30000, // Check every 30s
    retry: 1, // Only retry once for health checks
  });
}

/**
 * Hook to list all available LangFlow flows
 */
export function useLangFlowFlows() {
  return useQuery({
    queryKey: langflowKeys.flows(),
    queryFn: langflowApi.listFlows,
    staleTime: 60000, // Consider data fresh for 1 minute
  });
}

/**
 * Hook to get a specific LangFlow flow by ID
 */
export function useLangFlowFlow(id: string) {
  return useQuery({
    queryKey: langflowKeys.flow(id),
    queryFn: () => langflowApi.getFlow(id),
    enabled: !!id, // Only run query if ID is provided
    staleTime: 60000, // Consider data fresh for 1 minute
  });
}
