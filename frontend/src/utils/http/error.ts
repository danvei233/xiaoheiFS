/**
 * HTTP 错误处理模块
 *
 * 提供统一的 HTTP 请求错误处理机制
 *
 * ## 主要功能
 *
 * - 自定义 HttpError 错误类，封装错误信息、状态码、时间戳等
 * - 错误拦截和转换，将 Axios 错误转换为标准的 HttpError
 * - 错误消息国际化处理，根据状态码返回对应的多语言错误提示
 * - 错误日志记录，便于问题追踪和调试
 * - 错误和成功消息的统一展示
 * - 类型守卫函数，用于判断错误类型
 *
 * ## 使用场景
 *
 * - HTTP 请求拦截器中统一处理错误
 * - 业务代码中捕获和处理特定错误
 * - 错误日志收集和上报
 *
 * @module utils/http/error
 * @author Art Design Pro Team
 */
import { AxiosError } from 'axios'
import { ApiStatus } from './status'
import { $t } from '@/locales'

// 错误响应接口
export interface ErrorResponse {
  /** 错误状态码 */
  code: number
  /** 错误消息 */
  msg: string
  /** 错误附加数据 */
  data?: unknown
}

// 错误日志数据接口
export interface ErrorLogData {
  /** 错误状态码 */
  code: number
  /** 错误消息 */
  message: string
  /** 错误附加数据 */
  data?: unknown
  /** 错误发生时间戳 */
  timestamp: string
  /** 请求 URL */
  url?: string
  /** 请求方法 */
  method?: string
  /** 错误堆栈信息 */
  stack?: string
}

// 自定义 HttpError 类
export class HttpError extends Error {
  public readonly code: number
  public readonly data?: unknown
  public readonly timestamp: string
  public readonly url?: string
  public readonly method?: string
  /** 请求是否被调用方主动取消（AbortController / ERR_CANCELED），取消不应弹错误 toast */
  public readonly canceled: boolean

  constructor(
    message: string,
    code: number,
    options?: {
      data?: unknown
      url?: string
      method?: string
      canceled?: boolean
    }
  ) {
    super(message)
    this.name = 'HttpError'
    this.code = code
    this.data = options?.data
    this.timestamp = new Date().toISOString()
    this.url = options?.url
    this.method = options?.method
    this.canceled = options?.canceled ?? false
  }

  public toLogData(): ErrorLogData {
    return {
      code: this.code,
      message: this.message,
      data: this.data,
      timestamp: this.timestamp,
      url: this.url,
      method: this.method,
      stack: this.stack
    }
  }
}

/**
 * 获取错误消息
 * @param status 错误状态码
 * @returns 错误消息
 */
const getErrorMessage = (status: number): string => {
  const errorMap: Record<number, string> = {
    [ApiStatus.unauthorized]: 'httpMsg.unauthorized',
    [ApiStatus.forbidden]: 'httpMsg.forbidden',
    [ApiStatus.notFound]: 'httpMsg.notFound',
    [ApiStatus.methodNotAllowed]: 'httpMsg.methodNotAllowed',
    [ApiStatus.requestTimeout]: 'httpMsg.requestTimeout',
    [ApiStatus.internalServerError]: 'httpMsg.internalServerError',
    [ApiStatus.badGateway]: 'httpMsg.badGateway',
    [ApiStatus.serviceUnavailable]: 'httpMsg.serviceUnavailable',
    [ApiStatus.gatewayTimeout]: 'httpMsg.gatewayTimeout'
  }

  return $t(errorMap[status] || 'httpMsg.internalServerError')
}

/**
 * 解析错误消息的优先级（纯函数，便于回归测试）：
 *
 *  1. 服务端返回的 `data.msg` / `data.error` / `data.message` —— 业务明细
 *     （如"余额不足"）必须透传；兼容后端不同错误字段（H6）；
 *  2. 按状态码查本地化文案 —— 服务端无 `msg` 时（nginx 502/504 返回 HTML、
 *     后端遗漏 msg），显示 `httpMsg.badGateway` 这类本地化文案，而不是
 *     axios 的英文原始消息（"Request failed with status code 502"）；
 *  3. axios 原始 `error.message` —— 网络层语义信息；
 *  4. 通用兜底。
 *
 * 这个顺序在 m054 L5 修复（服务端明细透传）与"无 msg 时不应回退到英文原文"
 * 之间取得平衡。见 verify-r2 R-9。
 */

/** 从错误响应体提取服务端消息：兼容 `{msg}` / `{error}` / `{message}`（H6）。 */
export function resolveServerMessage(data: unknown): string {
  if (!data || typeof data !== 'object') return ''
  const record = data as Record<string, unknown>
  const value = record.msg ?? record.error ?? record.message
  return typeof value === 'string' ? value : ''
}
export const resolveErrorMessage = (
  dataMsg: string | undefined,
  statusCode: number | undefined,
  axiosMessage: string | undefined
): string => {
  if (dataMsg) return dataMsg
  if (statusCode) return getErrorMessage(statusCode)
  if (axiosMessage) return axiosMessage
  return $t('httpMsg.requestFailed')
}

/**
 * 处理错误
 * @param error 错误对象
 * @returns 错误对象
 */
export function handleError(error: AxiosError<ErrorResponse>): never {
  // 处理取消的请求
  if (error.code === 'ERR_CANCELED') {
    throw new HttpError($t('httpMsg.requestCancelled'), ApiStatus.error, { canceled: true })
  }

  const statusCode = error.response?.status
  const requestConfig = error.config

  // 处理网络错误
  if (!error.response) {
    throw new HttpError($t('httpMsg.networkError'), ApiStatus.error, {
      url: requestConfig?.url,
      method: requestConfig?.method?.toUpperCase()
    })
  }

  // 处理 HTTP 状态码错误
  const data = error.response.data
  const message = resolveErrorMessage(resolveServerMessage(data), statusCode, error.message)
  throw new HttpError(message, statusCode || ApiStatus.error, {
    data,
    url: requestConfig?.url,
    method: requestConfig?.method?.toUpperCase()
  })
}

/**
 * 显示错误消息
 * @param error 错误对象
 * @param showMessage 是否显示错误消息
 */
export function showError(error: HttpError, showMessage: boolean = true): void {
  if (showMessage) {
    ElMessage.error(error.message)
  }
  // 记录错误日志
  console.error('[HTTP Error]', error.toLogData())
}

/**
 * 显示成功消息
 * @param message 成功消息
 * @param showMessage 是否显示消息
 */
export function showSuccess(message: string, showMessage: boolean = true): void {
  if (showMessage) {
    ElMessage.success(message)
  }
}

/**
 * 判断是否为 HttpError 类型
 * @param error 错误对象
 * @returns 是否为 HttpError 类型
 */
export const isHttpError = (error: unknown): error is HttpError => {
  return error instanceof HttpError
}
