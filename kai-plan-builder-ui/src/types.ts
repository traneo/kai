export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
}

export interface ChatResponse {
  conversation_id: string
  reply: string
  spec: string
  spec_updated: boolean
  suggest_continue: boolean
}

export interface GenerateResponse {
  conversation_id: string
  yaml: string
}

export type Page = 'builder' | 'review'

export interface StepPolicy {
  allowedDirs: string[]
  agent: string
  allowedTools: string[]
  allowedCommands: string[]
  maxRetries: number
  timeoutSeconds: number
  retryDelaySeconds: number
  retryBackoff: string
  saveState: boolean
}

export interface BuilderStep {
  id: string
  prompt: string
  dependsOn: string[]
  validation: string[]
  approval: string
  policy: StepPolicy
}

export interface PipelineConfig {
  project: string
  repoURL: string
  repoBaseBranch: string
  repoProvider: string
  repoTokenRef: string
  outputType: string
  branchPrefix: string
  steps: BuilderStep[]
}

export const VALIDATION_GATES = ['exit_zero', 'lint', 'typecheck', 'tests', 'diff_review']
