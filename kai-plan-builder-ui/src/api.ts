import type { ChatResponse, GenerateResponse } from './types'

const API = '/api/v1/plan-builder'

export async function chat(conversationId: string | null, message: string): Promise<ChatResponse> {
  const res = await fetch(`${API}/chat`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      conversation_id: conversationId || undefined,
      message,
    }),
  })
  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.error || 'chat request failed')
  }
  return res.json()
}

export async function generatePipeline(conversationId: string): Promise<GenerateResponse> {
  const res = await fetch(`${API}/generate-pipeline`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ conversation_id: conversationId }),
  })
  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.error || 'pipeline generation failed')
  }
  return res.json()
}

export async function createPipeline(yaml: string): Promise<{ id: string }> {
  const res = await fetch('/api/pipelines', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ yaml }),
  })
  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.error || 'failed to create pipeline')
  }
  return res.json()
}
