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
