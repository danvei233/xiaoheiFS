import request from '@/utils/http'

export type InstallDBType = 'sqlite' | 'mysql'

export interface InstallDBConfig {
  type: InstallDBType
  path?: string
  dsn?: string
}

export interface InstallStatusResponse {
  installed?: boolean
}

export interface InstallDBCheckPayload {
  db: InstallDBConfig
}

export interface InstallDBCheckResponse {
  ok?: boolean
  error?: string
}

export interface InstallPayload {
  db: InstallDBConfig
  site: {
    name: string
    url?: string
    admin_path: string
  }
  admin: {
    username: string
    password: string
  }
}

export interface InstallRunResponse {
  ok?: boolean
  restart_required?: boolean
  config_file?: string
}

export interface InstallValidateAdminPathResponse {
  valid?: boolean
}

export function fetchInstallStatus() {
  return request.get<InstallStatusResponse>({
    url: '/api/v1/install/status',
    showErrorMessage: false
  })
}

export function checkInstallDB(payload: InstallDBCheckPayload) {
  return request.post<InstallDBCheckResponse>({
    url: '/api/v1/install/db/check',
    data: payload,
    showErrorMessage: false
  })
}

export function runInstall(payload: InstallPayload) {
  return request.post<InstallRunResponse>({
    url: '/api/v1/install',
    data: payload,
    showErrorMessage: false
  })
}

export function validateInstallAdminPath(path: string) {
  return request.post<InstallValidateAdminPathResponse>({
    url: '/api/v1/install/validate-admin-path',
    data: { path },
    showErrorMessage: false
  })
}
