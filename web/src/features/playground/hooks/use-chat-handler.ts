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
import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { createVideoGeneration, fetchVideoTask, sendChatCompletion } from '../api'
import { ERROR_MESSAGES, MESSAGE_ROLES, MESSAGE_STATUS } from '../constants'
import {
  applyStreamingChunk,
  buildChatCompletionPayload,
  updateAssistantMessageByKey,
  updateAssistantMessageWithError,
  updateLastAssistantMessage,
  parseRequestErrorDetails,
  applyChatCompletionResponse,
  completeAssistantMessage,
  hasChatCompletionChoice,
  isAssistantMessageFinal,
  isAssistantMessagePending,
  isSeedanceModel,
  getMessageContent,
  getVideoProgressLabel,
  isVideoTaskFailed,
  pollVideoTaskUntilDone,
  resolveVideoPlaybackUrl,
  updateCurrentVersionContent,
} from '../lib'
import type { Message, OpenAIVideoTask, PlaygroundConfig, ParameterEnabled } from '../types'
import { useStreamRequest } from './use-stream-request'

interface UseChatHandlerOptions {
  config: PlaygroundConfig
  parameterEnabled: ParameterEnabled
  messages: Message[]
  isLoadingMessages: boolean
  onMessageUpdate: (updater: (prev: Message[]) => Message[]) => void
}

const KNOWN_ERROR_MESSAGES = new Set<string>(Object.values(ERROR_MESSAGES))
const STREAM_UPDATE_FLUSH_MS = 50

type PendingStreamChunks = {
  generation: number
  content: string
  reasoning: string
}

function mergePendingStreamChunk(
  currentChunk: string,
  nextChunk: string
): string {
  if (!currentChunk || !nextChunk.startsWith(currentChunk)) {
    return currentChunk + nextChunk
  }

  return nextChunk
}

/**
 * Hook for handling chat message sending and receiving
 */
export function useChatHandler({
  config,
  parameterEnabled,
  messages,
  isLoadingMessages,
  onMessageUpdate,
}: UseChatHandlerOptions) {
  const { t } = useTranslation()
  const { sendStreamRequest, stopStream, isStreaming } = useStreamRequest()
  const [isRequesting, setIsRequesting] = useState(false)
  const abortControllerRef = useRef<AbortController | null>(null)
  const requestGenerationRef = useRef(0)
  const didResumeVideoTasksRef = useRef(false)
  const pendingStreamChunksRef = useRef<PendingStreamChunks>({
    generation: 0,
    content: '',
    reasoning: '',
  })
  const streamFlushTimerRef = useRef<number | null>(null)

  const discardPendingStreamUpdates = useCallback((generation: number) => {
    if (streamFlushTimerRef.current !== null) {
      window.clearTimeout(streamFlushTimerRef.current)
      streamFlushTimerRef.current = null
    }
    pendingStreamChunksRef.current = {
      generation,
      content: '',
      reasoning: '',
    }
  }, [])

  const flushStreamUpdates = useCallback(
    (generation: number) => {
      if (generation !== requestGenerationRef.current) return
      if (streamFlushTimerRef.current !== null) {
        window.clearTimeout(streamFlushTimerRef.current)
        streamFlushTimerRef.current = null
      }

      const pendingChunks = pendingStreamChunksRef.current
      if (pendingChunks.generation !== generation) return
      if (!pendingChunks.reasoning && !pendingChunks.content) {
        return
      }

      pendingStreamChunksRef.current = {
        generation,
        content: '',
        reasoning: '',
      }
      onMessageUpdate((prev) => {
        if (generation !== requestGenerationRef.current) return prev
        return updateLastAssistantMessage(prev, (message) => {
          let updatedMessage = message

          if (pendingChunks.reasoning) {
            updatedMessage = applyStreamingChunk(
              updatedMessage,
              'reasoning',
              pendingChunks.reasoning
            )
          }

          if (pendingChunks.content) {
            updatedMessage = applyStreamingChunk(
              updatedMessage,
              'content',
              pendingChunks.content
            )
          }

          return updatedMessage
        })
      })
    },
    [onMessageUpdate]
  )

  const scheduleStreamFlush = useCallback(
    (generation: number) => {
      if (generation !== requestGenerationRef.current) return
      if (streamFlushTimerRef.current !== null) {
        return
      }

      streamFlushTimerRef.current = window.setTimeout(() => {
        flushStreamUpdates(generation)
      }, STREAM_UPDATE_FLUSH_MS)
    },
    [flushStreamUpdates]
  )

  useEffect(
    () => () => {
      requestGenerationRef.current += 1
      if (streamFlushTimerRef.current !== null) {
        window.clearTimeout(streamFlushTimerRef.current)
      }
      abortControllerRef.current?.abort()
      abortControllerRef.current = null
    },
    []
  )

  const getDisplayError = useCallback(
    (error: string) => {
      if (KNOWN_ERROR_MESSAGES.has(error)) {
        return t(error)
      }

      const connectionClosedSuffix = `: ${ERROR_MESSAGES.CONNECTION_CLOSED}`
      if (error.endsWith(connectionClosedSuffix)) {
        return `${error.slice(0, -ERROR_MESSAGES.CONNECTION_CLOSED.length)}${t(
          ERROR_MESSAGES.CONNECTION_CLOSED
        )}`
      }

      return error
    },
    [t]
  )

  // Handle stream update
  const handleStreamUpdate = useCallback(
    (generation: number, type: 'reasoning' | 'content', chunk: string) => {
      if (generation !== requestGenerationRef.current) return
      if (pendingStreamChunksRef.current.generation !== generation) return
      pendingStreamChunksRef.current[type] = mergePendingStreamChunk(
        pendingStreamChunksRef.current[type],
        chunk
      )
      scheduleStreamFlush(generation)
    },
    [scheduleStreamFlush]
  )

  // Handle stream complete
  const handleStreamComplete = useCallback(
    (generation: number) => {
      if (generation !== requestGenerationRef.current) return
      flushStreamUpdates(generation)
      setIsRequesting(false)
      onMessageUpdate((prev) => {
        if (generation !== requestGenerationRef.current) return prev
        return updateLastAssistantMessage(prev, (message) =>
          isAssistantMessageFinal(message)
            ? message
            : completeAssistantMessage(message)
        )
      })
    },
    [flushStreamUpdates, onMessageUpdate]
  )

  // Handle stream error
  const handleStreamError = useCallback(
    (generation: number, error: string, errorCode?: string) => {
      if (generation !== requestGenerationRef.current) return
      flushStreamUpdates(generation)
      setIsRequesting(false)
      const displayError = getDisplayError(error)
      toast.error(displayError)
      const errorTitle = t(ERROR_MESSAGES.API_REQUEST_ERROR)
      onMessageUpdate((prev) => {
        if (generation !== requestGenerationRef.current) return prev
        return updateAssistantMessageWithError(
          prev,
          displayError,
          errorCode,
          errorTitle
        )
      })
    },
    [flushStreamUpdates, getDisplayError, onMessageUpdate, t]
  )

  // Send streaming chat request
  const sendStreamingChat = useCallback(
    (messages: Message[]) => {
      const generation = requestGenerationRef.current + 1
      requestGenerationRef.current = generation
      abortControllerRef.current?.abort()
      abortControllerRef.current = null
      discardPendingStreamUpdates(generation)
      setIsRequesting(true)
      const payload = buildChatCompletionPayload(
        messages,
        config,
        parameterEnabled
      )
      void sendStreamRequest(
        payload,
        (type, chunk) => handleStreamUpdate(generation, type, chunk),
        () => handleStreamComplete(generation),
        (error, errorCode) => handleStreamError(generation, error, errorCode)
      )
    },
    [
      config,
      parameterEnabled,
      sendStreamRequest,
      discardPendingStreamUpdates,
      handleStreamUpdate,
      handleStreamComplete,
      handleStreamError,
    ]
  )

  // Send non-streaming chat request
  const sendNonStreamingChat = useCallback(
    async (messages: Message[]) => {
      const payload = buildChatCompletionPayload(
        messages,
        config,
        parameterEnabled
      )
      const generation = requestGenerationRef.current + 1
      const abortController = new AbortController()

      requestGenerationRef.current = generation
      stopStream()
      discardPendingStreamUpdates(generation)
      abortControllerRef.current?.abort()
      abortControllerRef.current = abortController

      try {
        setIsRequesting(true)
        const response = await sendChatCompletion(
          payload,
          abortController.signal
        )
        if (
          abortController.signal.aborted ||
          requestGenerationRef.current !== generation
        ) {
          return
        }

        if (!hasChatCompletionChoice(response)) {
          handleStreamError(generation, ERROR_MESSAGES.API_REQUEST_ERROR)
          return
        }

        onMessageUpdate((prev) => {
          if (requestGenerationRef.current !== generation) return prev
          return updateLastAssistantMessage(prev, (message) => {
            const updatedMessage = applyChatCompletionResponse(
              message,
              response
            )

            return updatedMessage ?? message
          })
        })
      } catch (error: unknown) {
        if (
          abortController.signal.aborted ||
          requestGenerationRef.current !== generation
        ) {
          return
        }

        const { errorCode, errorMessage } = parseRequestErrorDetails(error)
        handleStreamError(generation, errorMessage, errorCode)
      } finally {
        if (requestGenerationRef.current === generation) {
          abortControllerRef.current = null
          setIsRequesting(false)
        }
      }
    },
    [
      config,
      parameterEnabled,
      stopStream,
      discardPendingStreamUpdates,
      onMessageUpdate,
      handleStreamError,
    ]
  )

  // Poll a video task and write progress / result onto a specific assistant message.
  const pollVideoTaskForMessage = useCallback(
    async (
      generation: number,
      messageKey: string,
      taskId: string,
      signal: AbortSignal
    ) => {
      const finalTask = await pollVideoTaskUntilDone(fetchVideoTask, taskId, {
        signal,
        onProgress: (task: OpenAIVideoTask) => {
          if (requestGenerationRef.current !== generation) return
          const label = getVideoProgressLabel(task.status, task.progress)
          onMessageUpdate((prev) =>
            updateAssistantMessageByKey(prev, messageKey, (message) => ({
              ...updateCurrentVersionContent(
                message,
                `${t(ERROR_MESSAGES.VIDEO_GENERATING)} (${label})`
              ),
              status: MESSAGE_STATUS.STREAMING,
              videoTaskId: taskId,
              videoUrl: null,
            }))
          )
        },
      })

      if (signal.aborted || requestGenerationRef.current !== generation) {
        return
      }

      if (isVideoTaskFailed(finalTask.status)) {
        handleStreamError(
          generation,
          finalTask.error?.message || ERROR_MESSAGES.VIDEO_FAILED,
          finalTask.error?.code
        )
        return
      }

      const videoUrl = resolveVideoPlaybackUrl(finalTask, taskId)
      onMessageUpdate((prev) => {
        if (requestGenerationRef.current !== generation) return prev
        return updateAssistantMessageByKey(prev, messageKey, (message) =>
          completeAssistantMessage({
            ...updateCurrentVersionContent(message, ''),
            videoUrl,
            videoTaskId: taskId,
          })
        )
      })
    },
    [handleStreamError, onMessageUpdate, t]
  )

  // Resume incomplete video generations restored from localStorage.
  const resumeIncompleteVideoTasks = useCallback(() => {
    const incomplete = [...messages]
      .reverse()
      .find(
        (message) =>
          message.from === MESSAGE_ROLES.ASSISTANT &&
          Boolean(message.videoTaskId) &&
          !message.videoUrl
      )

    if (!incomplete?.videoTaskId) {
      return
    }

    const generation = requestGenerationRef.current + 1
    const abortController = new AbortController()
    const taskId = incomplete.videoTaskId
    const messageKey = incomplete.key

    requestGenerationRef.current = generation
    stopStream()
    discardPendingStreamUpdates(generation)
    abortControllerRef.current?.abort()
    abortControllerRef.current = abortController

    void (async () => {
      try {
        setIsRequesting(true)
        onMessageUpdate((prev) =>
          updateAssistantMessageByKey(prev, messageKey, (message) => ({
            ...message,
            status: MESSAGE_STATUS.STREAMING,
            videoUrl: null,
            videoTaskId: taskId,
          }))
        )
        await pollVideoTaskForMessage(
          generation,
          messageKey,
          taskId,
          abortController.signal
        )
      } catch (error: unknown) {
        if (
          abortController.signal.aborted ||
          requestGenerationRef.current !== generation
        ) {
          return
        }

        if (
          error instanceof Error &&
          error.message === 'Video generation timed out'
        ) {
          handleStreamError(generation, ERROR_MESSAGES.VIDEO_TIMEOUT)
          return
        }

        const { errorCode, errorMessage } = parseRequestErrorDetails(error)
        handleStreamError(generation, errorMessage, errorCode)
      } finally {
        if (requestGenerationRef.current === generation) {
          abortControllerRef.current = null
          setIsRequesting(false)
        }
      }
    })()
  }, [
    messages,
    stopStream,
    discardPendingStreamUpdates,
    onMessageUpdate,
    pollVideoTaskForMessage,
    handleStreamError,
  ])

  useEffect(() => {
    if (isLoadingMessages || didResumeVideoTasksRef.current) {
      return
    }
    didResumeVideoTasksRef.current = true
    resumeIncompleteVideoTasks()
  }, [isLoadingMessages, resumeIncompleteVideoTasks])

  // Send video generation request for Seedance models
  const sendVideoGenerationChat = useCallback(
    async (messagesForSend: Message[]) => {
      const generation = requestGenerationRef.current + 1
      const abortController = new AbortController()

      requestGenerationRef.current = generation
      stopStream()
      discardPendingStreamUpdates(generation)
      abortControllerRef.current?.abort()
      abortControllerRef.current = abortController

      const lastUserMessage = [...messagesForSend]
        .reverse()
        .find((message) => message.from === 'user')
      const prompt = lastUserMessage
        ? getMessageContent(lastUserMessage).trim()
        : ''

      if (!prompt) {
        handleStreamError(generation, ERROR_MESSAGES.VIDEO_PROMPT_REQUIRED)
        return
      }

      const assistantMessage = [...messagesForSend]
        .reverse()
        .find((message) => message.from === MESSAGE_ROLES.ASSISTANT)
      const messageKey = assistantMessage?.key
      if (!messageKey) {
        handleStreamError(generation, ERROR_MESSAGES.API_REQUEST_ERROR)
        return
      }

      try {
        setIsRequesting(true)
        onMessageUpdate((prev) => {
          if (requestGenerationRef.current !== generation) return prev
          return updateAssistantMessageByKey(prev, messageKey, (message) => ({
            ...updateCurrentVersionContent(
              message,
              t(ERROR_MESSAGES.VIDEO_GENERATING)
            ),
            status: MESSAGE_STATUS.STREAMING,
            videoUrl: null,
            videoTaskId: null,
          }))
        })

        const created = await createVideoGeneration(
          {
            model: config.model,
            prompt,
            group: config.group,
            duration: 5,
            size: '720p',
          },
          abortController.signal
        )

        if (
          abortController.signal.aborted ||
          requestGenerationRef.current !== generation
        ) {
          return
        }

        const taskId = created.id || created.task_id
        if (!taskId) {
          handleStreamError(generation, ERROR_MESSAGES.API_REQUEST_ERROR)
          return
        }

        onMessageUpdate((prev) => {
          if (requestGenerationRef.current !== generation) return prev
          return updateAssistantMessageByKey(prev, messageKey, (message) => ({
            ...message,
            videoTaskId: taskId,
          }))
        })

        await pollVideoTaskForMessage(
          generation,
          messageKey,
          taskId,
          abortController.signal
        )
      } catch (error: unknown) {
        if (
          abortController.signal.aborted ||
          requestGenerationRef.current !== generation
        ) {
          return
        }

        if (
          error instanceof Error &&
          error.message === 'Video generation timed out'
        ) {
          handleStreamError(generation, ERROR_MESSAGES.VIDEO_TIMEOUT)
          return
        }

        const { errorCode, errorMessage } = parseRequestErrorDetails(error)
        handleStreamError(generation, errorMessage, errorCode)
      } finally {
        if (requestGenerationRef.current === generation) {
          abortControllerRef.current = null
          setIsRequesting(false)
        }
      }
    },
    [
      config.model,
      config.group,
      stopStream,
      discardPendingStreamUpdates,
      onMessageUpdate,
      handleStreamError,
      pollVideoTaskForMessage,
      t,
    ]
  )

  // Send chat request (stream or non-stream based on config)
  const sendChat = useCallback(
    (messages: Message[]) => {
      if (isSeedanceModel(config.model)) {
        void sendVideoGenerationChat(messages)
        return
      }
      if (config.stream) {
        sendStreamingChat(messages)
      } else {
        sendNonStreamingChat(messages)
      }
    },
    [
      config.stream,
      config.model,
      sendStreamingChat,
      sendNonStreamingChat,
      sendVideoGenerationChat,
    ]
  )

  // Stop generation
  const stopGeneration = useCallback(() => {
    const stoppedGeneration = requestGenerationRef.current
    flushStreamUpdates(stoppedGeneration)
    const idleGeneration = stoppedGeneration + 1
    requestGenerationRef.current = idleGeneration
    discardPendingStreamUpdates(idleGeneration)
    stopStream()
    abortControllerRef.current?.abort()
    abortControllerRef.current = null
    setIsRequesting(false)
    onMessageUpdate((prev) => {
      if (requestGenerationRef.current !== idleGeneration) return prev
      return updateLastAssistantMessage(prev, (message) =>
        isAssistantMessagePending(message)
          ? completeAssistantMessage(message)
          : message
      )
    })
  }, [
    stopStream,
    flushStreamUpdates,
    discardPendingStreamUpdates,
    onMessageUpdate,
  ])

  return {
    sendChat,
    stopGeneration,
    isGenerating: isStreaming || isRequesting,
  }
}
