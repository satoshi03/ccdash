export interface Settings {
  plan: 'Pro' | 'Max5' | 'Max20'
  timezone: string
  autoRefreshInterval: number // 秒単位
}

export const DEFAULT_SETTINGS: Settings = {
  plan: 'Pro',
  timezone: 'Asia/Tokyo',
  autoRefreshInterval: 60 // 1分間隔
}

export const PLAN_LIMITS = {
  Pro: 7000,
  Max5: 35000,
  Max20: 140000
}

export const TIMEZONE_OPTIONS = [
  { value: 'Asia/Tokyo', label: 'Asia/Tokyo (JST)' },
  { value: 'America/New_York', label: 'America/New_York (EST)' },
  { value: 'America/Los_Angeles', label: 'America/Los_Angeles (PST)' },
  { value: 'Europe/London', label: 'Europe/London (GMT)' },
  { value: 'UTC', label: 'UTC' }
]

export const REFRESH_INTERVAL_OPTIONS = [
  { value: 30, label: '30秒' },
  { value: 60, label: '1分' },
  { value: 300, label: '5分' },
  { value: 600, label: '10分' },
  { value: 1800, label: '30分' },
  { value: 3600, label: '1時間' }
]

export function getSettings(): Settings {
  if (typeof window === 'undefined') {
    return DEFAULT_SETTINGS
  }

  try {
    const stored = localStorage.getItem('claudeee-settings')
    if (stored) {
      return { ...DEFAULT_SETTINGS, ...JSON.parse(stored) }
    }
  } catch (error) {
    console.error('Failed to load settings:', error)
  }

  return DEFAULT_SETTINGS
}

export function saveSettings(settings: Settings): void {
  if (typeof window === 'undefined') {
    return
  }

  try {
    localStorage.setItem('claudeee-settings', JSON.stringify(settings))
  } catch (error) {
    console.error('Failed to save settings:', error)
  }
}