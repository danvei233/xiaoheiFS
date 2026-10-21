import {
  fetchAdminLogin,
  fetchAdminProfile,
  mapAdminProfileToUserInfo,
  type AdminAuthUserInfo,
  type AdminLoginParams,
  type AdminLoginResponse
} from './admin'

export function fetchLogin(params: AdminLoginParams) {
  return fetchAdminLogin(params)
}

export function fetchGetUserInfo(): Promise<AdminAuthUserInfo> {
  return fetchAdminProfile().then((profile) => mapAdminProfileToUserInfo(profile))
}

export type { AdminAuthUserInfo, AdminLoginParams, AdminLoginResponse }
