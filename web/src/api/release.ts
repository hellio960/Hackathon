import { apiClient } from './client'
import {
  NodeReleaseCreateReq,
  NodeReleaseContinueReq,
  NodeReleaseFilterSwitchReq,
  NodeReleaseListReq,
  NodeReleaseListResp,
  NodeReleaseDetail,
  NodeReleasePackagesReq,
  NodeReleasePackagesResp,
  NodeReleaseHistoryResp,
} from './types'

export const releaseApi = {
  createRelease: (data: NodeReleaseCreateReq) => {
    return apiClient.post('/release/create', data)
  },

  continueRelease: (releaseID: string, data: NodeReleaseContinueReq) => {
    return apiClient.post(`/release/${releaseID}/continue`, data)
  },

  rollbackRelease: (releaseID: string) => {
    return apiClient.post(`/release/${releaseID}/rollback`, {})
  },

  completeRelease: (releaseID: string) => {
    return apiClient.post(`/release/${releaseID}/complete`, {})
  },

  filterSwitch: (releaseID: string, data: NodeReleaseFilterSwitchReq) => {
    return apiClient.post(`/release/${releaseID}/filterswitch`, data)
  },

  getReleaseList: (data: NodeReleaseListReq) => {
    return apiClient.post<NodeReleaseListResp>('/release/list', data)
  },

  getReleaseDetail: (releaseID: string) => {
    return apiClient.get<NodeReleaseDetail>(`/release/${releaseID}/detail`)
  },

  exportAllowNodes: (releaseID: string) => {
    return apiClient.get<{ nodes: string[] }>(`/release/${releaseID}/allownodes/export`)
  },

  getReleaseHistory: (releaseID: string) => {
    return apiClient.get<NodeReleaseHistoryResp>(`/release/${releaseID}/history`)
  },

  getPackages: (data: NodeReleasePackagesReq) => {
    return apiClient.post<NodeReleasePackagesResp>('/release/packages', data)
  },
}
