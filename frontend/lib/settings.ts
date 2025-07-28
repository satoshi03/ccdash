export interface Settings {
  timezone: string
  autoRefreshInterval: number // 秒単位
  usageMode: 'p90_prediction' | 'fixed_limits'
  fixedLimits: {
    tokenLimit: number
    messageLimit: number
    costLimit: number
  }
}

export const DEFAULT_SETTINGS: Settings = {
  timezone: 'Asia/Tokyo',
  autoRefreshInterval: 60, // 1分間隔
  usageMode: 'p90_prediction',
  fixedLimits: {
    tokenLimit: 19000, // Pro plan default from Claude-Usage-Monitor
    messageLimit: 250,
    costLimit: 18.0
  }
}


// Fixed limits based on Claude-Usage-Monitor plans
export const FIXED_LIMITS_PRESETS = {
  Pro: {
    tokenLimit: 19000,
    messageLimit: 250,
    costLimit: 18.0
  },
  Max5: {
    tokenLimit: 88000,
    messageLimit: 1000,
    costLimit: 35.0
  },
  Max20: {
    tokenLimit: 220000,
    messageLimit: 2000,
    costLimit: 140.0
  },
  Custom: {
    tokenLimit: 44000,
    messageLimit: 250,
    costLimit: 50.0
  }
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
    const stored = localStorage.getItem('ccdash-settings')
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
    localStorage.setItem('ccdash-settings', JSON.stringify(settings))
  } catch (error) {
    console.error('Failed to save settings:', error)
  }
}