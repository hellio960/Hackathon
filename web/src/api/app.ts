import { apiClient } from './client'
import {
  NodeReleaseAllowAppsListResp,
  NodeReleaseAllowAppsUpdateReq,
} from './types'

export const appApi = {
  getAllowAppsList: (params?: { nodeType?: string; app?: string }) => {
    return apiClient.get<NodeReleaseAllowAppsListResp>('/release/allowapps', { params })
  },

  updateAllowApps: (data: NodeReleaseAllowAppsUpdateReq) => {
    return apiClient.post('/release/allowapps', data)
  },
}
