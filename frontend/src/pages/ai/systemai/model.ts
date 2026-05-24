export interface AIConfig {
  provider_id: string
  name: string
  base_url: string
  organization: string
  models: string[]
  default_model: string
  temperature: number
  timeout_seconds: number
  max_tokens: number
  purposes: string[]
  primary_for: string[]
  enabled: boolean
  has_secret: boolean
  updated_at: string
}
