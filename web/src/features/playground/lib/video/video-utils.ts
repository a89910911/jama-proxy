/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import {
  API_ENDPOINTS,
  VIDEO_POLL_INTERVAL_MS,
  VIDEO_POLL_MAX_ATTEMPTS,
} from '../../constants'
import type { OpenAIVideoTask } from '../../types'

function sleep(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    if (signal?.aborted) {
      reject(new DOMException('Aborted', 'AbortError'))
      return
    }
    const timer = window.setTimeout(() => {
      signal?.removeEventListener('abort', onAbort)
      resolve()
    }, ms)
    const onAbort = () => {
      window.clearTimeout(timer)
      reject(new DOMException('Aborted', 'AbortError'))
    }
    signal?.addEventListener('abort', onAbort, { once: true })
  })
}

export function getVideoContentUrl(taskId: string): string {
  return `${API_ENDPOINTS.VIDEO_CONTENT}/${taskId}/content`
}

export function resolveVideoPlaybackUrl(
  task: OpenAIVideoTask,
  taskId: string
): string {
  const metadataUrl = task.metadata?.url
  if (typeof metadataUrl === 'string' && metadataUrl.startsWith('http')) {
    return metadataUrl
  }
  return getVideoContentUrl(taskId)
}

export function isVideoTaskTerminal(status: string | undefined): boolean {
  const normalized = (status || '').toLowerCase()
  return (
    normalized === 'completed' ||
    normalized === 'failed' ||
    normalized === 'success' ||
    normalized === 'succeeded' ||
    normalized === 'failure'
  )
}

export function isVideoTaskFailed(status: string | undefined): boolean {
  const normalized = (status || '').toLowerCase()
  return normalized === 'failed' || normalized === 'failure'
}

export function getVideoProgressLabel(
  status: string | undefined,
  progress: number | undefined
): string {
  const pct =
    typeof progress === 'number' && progress > 0 ? ` ${progress}%` : ''
  switch ((status || '').toLowerCase()) {
    case 'queued':
    case 'submitted':
    case 'not_start':
    case 'unknown':
    case '':
      return `queued${pct}`
    case 'in_progress':
    case 'processing':
    case 'running':
      return `in progress${pct}`
    case 'completed':
    case 'success':
    case 'succeeded':
      return 'completed'
    case 'failed':
    case 'failure':
      return 'failed'
    default:
      return `queued${pct}`
  }
}

export async function pollVideoTaskUntilDone(
  fetchTask: (taskId: string, signal?: AbortSignal) => Promise<OpenAIVideoTask>,
  taskId: string,
  options: {
    signal?: AbortSignal
    onProgress?: (task: OpenAIVideoTask) => void
    intervalMs?: number
    maxAttempts?: number
  } = {}
): Promise<OpenAIVideoTask> {
  const intervalMs = options.intervalMs ?? VIDEO_POLL_INTERVAL_MS
  const maxAttempts = options.maxAttempts ?? VIDEO_POLL_MAX_ATTEMPTS

  for (let attempt = 0; attempt < maxAttempts; attempt++) {
    if (options.signal?.aborted) {
      throw new DOMException('Aborted', 'AbortError')
    }

    const task = await fetchTask(taskId, options.signal)
    options.onProgress?.(task)

    if (isVideoTaskTerminal(task.status)) {
      return task
    }

    await sleep(intervalMs, options.signal)
  }

  throw new Error('Video generation timed out')
}
