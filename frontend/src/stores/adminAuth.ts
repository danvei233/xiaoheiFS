import { defineStore } from 'pinia'
import { adminLogin, getAdminProfile } from '@/services/admin'
import type { AdminLoginResponse, AdminProfile } from '@/services/types'

const STORAGE_KEY = 'admin_token'
const MFA_STORAGE_KEY = 'admin_mfa_gate'

interface AdminMfaGateState {
  totpEnabled: boolean
  mfaRequired: boolean
  mfaBindRequired: boolean
  mfaUnlocked: boolean
}

const defaultMfaGateState = (): AdminMfaGateState => ({
  totpEnabled: false,
  mfaRequired: false,
  mfaBindRequired: false,
  mfaUnlocked: false
})

function loadMfaGateState(): AdminMfaGateState {
  try {
    const stored = JSON.parse(localStorage.getItem(MFA_STORAGE_KEY) || '{}')
    return {
      totpEnabled: stored.totpEnabled === true,
      mfaRequired: stored.mfaRequired === true,
      mfaBindRequired: stored.mfaBindRequired === true,
      mfaUnlocked: stored.mfaUnlocked === true
    }
  } catch {
    return defaultMfaGateState()
  }
}

function saveMfaGateState(state: AdminMfaGateState): void {
  try {
    localStorage.setItem(MFA_STORAGE_KEY, JSON.stringify(state))
  } catch {
    // Storage can be unavailable in hardened browser contexts; in-memory gating still applies.
  }
}

export const useAdminAuthStore = defineStore('adminAuth', {
  state: () => {
    const token = localStorage.getItem(STORAGE_KEY) || ''
    const mfaGate = token ? loadMfaGateState() : defaultMfaGateState()
    return {
      token,
      loading: false,
      profile: null as AdminProfile | null,
      ...mfaGate
    }
  },
  getters: {
    requiresMfa: (state): boolean =>
      (state.mfaRequired || state.mfaBindRequired) && !state.mfaUnlocked
  },
  actions: {
    async login(payload: {
      username: string
      password: string
      admin_path: string
    }): Promise<AdminLoginResponse> {
      this.loading = true
      try {
        const res = await adminLogin(payload)
        const data = res.data || {}
        const token = res.data?.access_token || ''
        this.setToken(token)
        // 登录响应里的 user 不是权限完整的 profile；MFA 完成后由路由守卫再拉取。
        this.profile = null
        this.setMfaGateState({
          totpEnabled: data.totp_enabled,
          mfaRequired: data.mfa_required,
          mfaBindRequired: data.mfa_bind_required,
          mfaUnlocked: data.mfa_unlocked
        })
        return data
      } finally {
        this.loading = false
      }
    },
    async fetchProfile() {
      const res = await getAdminProfile()
      this.profile = res.data || null
    },
    logout() {
      this.setToken('')
      this.profile = null
      this.totpEnabled = false
      this.mfaRequired = false
      this.mfaBindRequired = false
      this.mfaUnlocked = false
      localStorage.removeItem(MFA_STORAGE_KEY)
    },
    setToken(token: string) {
      this.token = token || ''
      if (this.token) {
        localStorage.setItem(STORAGE_KEY, this.token)
      } else {
        localStorage.removeItem(STORAGE_KEY)
      }
    },
    completeMfa(token: string) {
      this.setToken(token)
      this.profile = null
      this.setMfaGateState({
        totpEnabled: true,
        mfaRequired: false,
        mfaBindRequired: false,
        mfaUnlocked: true
      })
    },
    setMfaGateState(payload: Partial<AdminMfaGateState>) {
      if (typeof payload.mfaRequired === 'boolean') {
        this.mfaRequired = payload.mfaRequired
      }
      if (typeof payload.mfaBindRequired === 'boolean') {
        this.mfaBindRequired = payload.mfaBindRequired
      }
      if (typeof payload?.mfaUnlocked === 'boolean') {
        this.mfaUnlocked = payload.mfaUnlocked
      }
      if (typeof payload?.totpEnabled === 'boolean') {
        this.totpEnabled = payload.totpEnabled
      }
      saveMfaGateState({
        totpEnabled: this.totpEnabled,
        mfaRequired: this.mfaRequired,
        mfaBindRequired: this.mfaBindRequired,
        mfaUnlocked: this.mfaUnlocked
      })
    }
  }
})
