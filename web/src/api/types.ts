export interface AppConfig {
  url: string
  type: string
  cmd: string
  args?: string[]
  dir: string
  healthUrl?: string
  md5: string
}

export interface GrayFilter {
  devType?: string
  customerIds?: number[]
  status: 'online' | 'outline'
  stages: string[]
}

export interface StatInfo {
  totalCount: number
}

export interface GrayPolicy {
  nodeIds?: string[]
  filter?: GrayFilter
  percentage: number
  statInfo?: StatInfo
}

export interface NodeReleaseCreateReq {
  devType: string
  appName: string
  describe: string
  releaseType: 'formal' | 'beta'
  opType: 'add' | 'update' | 'delete'
  appConfig?: AppConfig
  grayPolicy: GrayPolicy
}

export interface NodeReleaseContinueReq {
  addNodes?: string[]
  delNodes?: string[]
  percentage: number
}

export interface NodeReleaseFilterSwitchReq {
  byNodeIds?: boolean
  byFilter?: GrayFilter
}

export interface NodeReleaseListReq {
  releaseIds?: string[]
  app?: string
  devTypes?: string[]
  releaseTypes?: string[]
  states?: string[]
  page: number
  size: number
}

export interface NodeReleaseBrief {
  id: string
  app: string
  deviceType: string
  releaseType: string
  state: string
  operator: string
  createAt: number
}

export interface NodeReleaseListResp {
  items: NodeReleaseBrief[]
  total: number
}

export interface NodeReleaseDetail {
  id: string
  app: string
  deviceType: string
  releaseType: string
  opType: string
  mainConfig?: AppConfig
  alterConfig?: AppConfig
  grayPolicy: GrayPolicy
  allowNodes: string[]
  allowNodesTotal: number
  state: string
  rollbackAllowed: boolean
  operator: string
  desc: string
  createAt: number
  updateAt: number
  endAt: number
}

export interface PackageInfo {
  url: string
  file: string
  md5: string
  size: string
}

export interface NodeReleasePackagesReq {
  devType: string
  app: string
  size?: number
  path?: string
}

export interface NodeReleasePackagesResp {
  packages: PackageInfo[]
}

export interface AllowApp {
  id: string
  name: string
  nodeType: string
  path: string
  operator: string
  createAt: number
  desc: string
}

export interface NodeReleaseAllowAppsListResp {
  apps: AllowApp[]
}

export interface NodeReleaseAllowAppsUpdateReq {
  operation: 'add' | 'del' | 'update'
  id?: string
  name?: string
  nodeTypes?: string[]
  desc?: string
  path?: string
}

export interface GrayPolicyRemark {
  nodeIdsAdd: string[]
  nodeIdsDel: string[]
  afterFilter?: GrayFilter
  filterChangeMode: string
  percentage: number
  afterPercentage: number
}

export interface AppConfigRemark {
  beforeMain?: AppConfig
  afterMain?: AppConfig
}

export interface NodeReleaseHistoryItem {
  operation: string
  operator: string
  opTime: number
  remark: string
  beforeState: string
  afterState: string
  grayPolicyInfo?: GrayPolicyRemark
  appConfigInfo?: AppConfigRemark
}

export interface NodeReleaseHistoryResp {
  items: NodeReleaseHistoryItem[]
}

export interface ApiResponse<T = any> {
  code: number
  msg: string
  data?: T
}
