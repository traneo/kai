const API = '/api'

export async function fetchAgents() {
  const res = await fetch(`${API}/agents`)
  if (!res.ok) throw new Error('failed to fetch agents')
  return res.json()
}

export async function fetchPipelines() {
  const res = await fetch(`${API}/pipelines`)
  if (!res.ok) throw new Error('failed to fetch pipelines')
  return res.json()
}

export async function fetchStatus() {
  const res = await fetch(`${API}/status`)
  if (!res.ok) throw new Error('failed to fetch status')
  return res.json()
}

export async function fetchStats() {
  const res = await fetch(`${API}/stats?runs=true`)
  if (!res.ok) throw new Error('failed to fetch stats')
  return res.json()
}

export async function createPipeline(yaml: string) {
  const res = await fetch(`${API}/pipelines`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ yaml })
  })
  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.error || 'failed to create pipeline')
  }
  return res.json()
}

export async function fetchPipelineDetail(id: string) {
  const res = await fetch(`${API}/pipelines/${id}`)
  if (!res.ok) throw new Error('failed to fetch pipeline detail')
  return res.json()
}

export async function approveStep(pipelineId: string, stepId: string, action: 'approve' | 'reject', message?: string) {
  const res = await fetch(`${API}/pipelines/${pipelineId}/steps/${stepId}/approve`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ action, message })
  })
  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.error || 'failed to approve step')
  }
  return res.json()
}

export async function fetchAuditLog(limit = 50, runId?: string) {
  const params = new URLSearchParams({ limit: String(limit) })
  if (runId) params.set('run_id', runId)
  const res = await fetch(`${API}/audit?${params}`)
  if (!res.ok) throw new Error('failed to fetch audit log')
  return res.json()
}

export async function retryStep(pipelineId: string, stepId: string) {
  const res = await fetch(`${API}/pipelines/${pipelineId}/steps/${stepId}/retry`, { method: 'POST' })
  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.error || 'failed to retry step')
  }
  return res.json()
}

export async function cancelPipeline(id: string) {
  const res = await fetch(`${API}/pipelines/${id}/cancel`, { method: 'POST' })
  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.error || 'failed to cancel pipeline')
  }
  return res.json()
}

export async function fetchPipelineYAML(id: string) {
  const res = await fetch(`${API}/pipelines/${id}/yaml`)
  if (!res.ok) throw new Error('failed to fetch pipeline YAML')
  return res.text()
}

export async function fetchPolicies() {
  const res = await fetch(`${API}/policies`)
  if (!res.ok) throw new Error('failed to fetch policies')
  return res.json()
}

export async function fetchQueue() {
  const res = await fetch(`${API}/queue`)
  if (!res.ok) throw new Error('failed to fetch queue')
  return res.json()
}

export async function fetchConversation(runId: string, stepId: string, limit = 500) {
  const res = await fetch(`${API}/pipelines/${runId}/steps/${stepId}/conversation?limit=${limit}`)
  if (!res.ok) throw new Error('failed to fetch conversation')
  return res.json()
}

export async function fetchSecrets() {
  const res = await fetch(`${API}/secrets`)
  if (!res.ok) throw new Error('failed to fetch secrets')
  return res.json()
}

export async function setSecret(name: string, value: string, description?: string) {
  const res = await fetch(`${API}/secrets`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, value, description })
  })
  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.error || 'failed to set secret')
  }
  return res.json()
}

export async function deleteSecret(name: string) {
  const res = await fetch(`${API}/secrets/${encodeURIComponent(name)}`, { method: 'DELETE' })
  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.error || 'failed to delete secret')
  }
  return res.json()
}

export async function fetchActiveConfig() {
  const res = await fetch(`${API}/v1/config`)
  if (!res.ok) throw new Error('failed to fetch config')
  return res.json()
}

export async function fetchConfigVersions(status?: string) {
  const params = status ? `?status=${status}` : ''
  const res = await fetch(`${API}/v1/config/versions${params}`)
  if (!res.ok) throw new Error('failed to fetch versions')
  return res.json()
}

export async function fetchConfigVersion(id: string) {
  const res = await fetch(`${API}/v1/config/versions/${id}`)
  if (!res.ok) throw new Error('failed to fetch version')
  return res.json()
}

export async function createConfigDraft() {
  const res = await fetch(`${API}/v1/config/versions`, { method: 'POST' })
  if (!res.ok) throw new Error('failed to create draft')
  return res.json()
}

export async function updateConfigDraft(id: string, config: unknown, message: string) {
  const res = await fetch(`${API}/v1/config/versions/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ config, message })
  })
  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.error || 'failed to update draft')
  }
  return res.json()
}

export async function publishConfigVersion(id: string) {
  const res = await fetch(`${API}/v1/config/versions/${id}/publish`, { method: 'POST' })
  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.error || 'failed to publish')
  }
  return res.json()
}

export async function activateConfigVersion(id: string) {
  const res = await fetch(`${API}/v1/config/versions/${id}/activate`, { method: 'POST' })
  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.error || 'failed to activate')
  }
  return res.json()
}

export async function rollbackConfigVersion(id: string) {
  const res = await fetch(`${API}/v1/config/versions/${id}/rollback`, { method: 'POST' })
  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.error || 'failed to rollback')
  }
  return res.json()
}

export async function fetchConfigStatus() {
  const res = await fetch(`${API}/v1/config/status`)
  if (!res.ok) throw new Error('failed to fetch config status')
  return res.json()
}

export function subscribeEvents(onEvent: (evt: unknown) => void): () => void {
  const es = new EventSource(`${API}/events`)

  es.onmessage = (e) => {
    try {
      const data = JSON.parse(e.data)
      onEvent(data)
    } catch { /* ignore parse errors */ }
  }

  // Don't close on error — browser auto-reconnects with backoff
  es.onerror = () => {}

  return () => es.close()
}
