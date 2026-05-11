import { apiClient, unwrap } from './client'
import type { ActivityEvent } from '../types/api'

export async function getMyActivity(limit = 20): Promise<ActivityEvent[]> {
  return unwrap(
    await apiClient.get<{ data: ActivityEvent[] }>('/user/activity', {
      params: { limit },
    })
  ) ?? []
}
