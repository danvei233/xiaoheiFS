export interface SseMessage {
  id?: string
  event?: string
  data: string
}

export interface SseOptions {
  headers?: Record<string, string>
  onMessage: (msg: SseMessage) => void
  onError?: (error: unknown) => void
  retryMs?: number
  maxRetryMs?: number
}

function parseEventChunk(chunk: string) {
  const lines = chunk.split('\n')
  const payload: SseMessage = { data: '' }

  lines.forEach((line) => {
    if (line.startsWith('id:')) {
      payload.id = line.replace('id:', '').trim()
      return
    }

    if (line.startsWith('event:')) {
      payload.event = line.replace('event:', '').trim()
      return
    }

    if (line.startsWith('data:')) {
      const data = line.replace('data:', '').trim()
      payload.data = payload.data ? `${payload.data}\n${data}` : data
    }
  })

  return payload.data ? payload : null
}

export function createSseConnection(url: string, options: SseOptions) {
  let closed = false
  let controller: AbortController | null = null
  let lastEventId = ''
  let retryMs = options.retryMs ?? 1500
  const maxRetryMs = options.maxRetryMs ?? 8000

  const connect = async () => {
    if (closed) {
      return
    }

    controller?.abort()
    controller = new AbortController()

    try {
      const headers: Record<string, string> = {
        Accept: 'text/event-stream',
        ...options.headers
      }

      if (lastEventId) {
        headers['Last-Event-ID'] = lastEventId
      }

      const response = await fetch(url, {
        headers,
        signal: controller.signal
      })

      if (!response.ok || !response.body) {
        throw new Error(`SSE failed: ${response.status}`)
      }

      retryMs = options.retryMs ?? 1500

      const reader = response.body.getReader()
      const decoder = new TextDecoder('utf-8')
      let buffer = ''

      while (!closed) {
        const { value, done } = await reader.read()
        if (done) {
          break
        }

        buffer += decoder.decode(value, { stream: true })
        const parts = buffer.split('\n\n')
        buffer = parts.pop() || ''

        parts.forEach((chunk) => {
          const msg = parseEventChunk(chunk)
          if (!msg) {
            return
          }

          if (msg.id) {
            lastEventId = msg.id
          }

          options.onMessage(msg)
        })
      }
    } catch (error) {
      if (!closed) {
        options.onError?.(error)
        retryMs = Math.min(retryMs * 1.4, maxRetryMs)
      }
    } finally {
      if (!closed) {
        setTimeout(connect, retryMs)
      }
    }
  }

  connect()

  return {
    close: () => {
      closed = true
      controller?.abort()
    }
  }
}
