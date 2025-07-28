"use client"

import { useState, useEffect } from "react"
import { Settings as SettingsIcon } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { Settings, getSettings, saveSettings, TIMEZONE_OPTIONS, REFRESH_INTERVAL_OPTIONS, FIXED_LIMITS_PRESETS } from "@/lib/settings"
import { useI18n } from "@/hooks/use-i18n"

interface SettingsModalProps {
  onSettingsChange?: (settings: Settings) => void
}

export function SettingsModal({ onSettingsChange }: SettingsModalProps) {
  const { t } = useI18n()
  const [settings, setSettings] = useState<Settings>(() => getSettings())
  const [isOpen, setIsOpen] = useState(false)
  const [hasChanges, setHasChanges] = useState(false)

  useEffect(() => {
    const savedSettings = getSettings()
    setSettings(savedSettings)
  }, [])


  const handleTimezoneChange = (value: string) => {
    const newSettings = { ...settings, timezone: value }
    setSettings(newSettings)
    setHasChanges(true)
  }

  const handleRefreshIntervalChange = (value: string) => {
    const newSettings = { ...settings, autoRefreshInterval: parseInt(value) }
    setSettings(newSettings)
    setHasChanges(true)
  }

  const handleUsageModeChange = (value: string) => {
    const newSettings = { ...settings, usageMode: value as Settings['usageMode'] }
    setSettings(newSettings)
    setHasChanges(true)
  }

  const handleFixedLimitChange = (field: keyof Settings['fixedLimits'], value: string) => {
    const numValue = field === 'costLimit' ? parseFloat(value) : parseInt(value)
    if (isNaN(numValue)) return
    
    const newSettings = {
      ...settings,
      fixedLimits: {
        ...settings.fixedLimits,
        [field]: numValue
      }
    }
    setSettings(newSettings)
    setHasChanges(true)
  }

  const applyPreset = (preset: keyof typeof FIXED_LIMITS_PRESETS) => {
    const newSettings = {
      ...settings,
      fixedLimits: { ...FIXED_LIMITS_PRESETS[preset] }
    }
    setSettings(newSettings)
    setHasChanges(true)
  }

  const handleSave = () => {
    saveSettings(settings)
    setHasChanges(false)
    setIsOpen(false)
    onSettingsChange?.(settings)
  }

  const handleCancel = () => {
    const savedSettings = getSettings()
    setSettings(savedSettings)
    setHasChanges(false)
    setIsOpen(false)
  }

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          <SettingsIcon className="w-4 h-4 mr-2" />
          {t('common.settings')}
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[500px] max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{t('settings.title')}</DialogTitle>
          <DialogDescription>
            {t('settings.description')}
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          {/* Usage Mode Selection */}
          <div className="grid grid-cols-4 items-center gap-4">
            <Label htmlFor="usage-mode" className="text-right">
              {t('settings.usageMode')}
            </Label>
            <Select value={settings.usageMode} onValueChange={handleUsageModeChange}>
              <SelectTrigger className="col-span-3">
                <SelectValue placeholder={t('settings.usageModePlaceholder')} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="p90_prediction">{t('settings.p90Prediction')}</SelectItem>
                <SelectItem value="fixed_limits">{t('settings.fixedLimits')}</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Fixed Limits Configuration */}
          {settings.usageMode === 'fixed_limits' && (
            <div className="space-y-3 p-3 border rounded-lg bg-gray-50">
              <div className="flex justify-between items-center">
                <Label className="text-sm font-medium">{t('settings.fixedLimitsConfig')}</Label>
                <Select onValueChange={(value) => applyPreset(value as keyof typeof FIXED_LIMITS_PRESETS)}>
                  <SelectTrigger className="w-32">
                    <SelectValue placeholder={t('settings.preset')} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="Pro">Pro</SelectItem>
                    <SelectItem value="Max5">Max5</SelectItem>
                    <SelectItem value="Max20">Max20</SelectItem>
                    <SelectItem value="Custom">Custom</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              
              <div className="grid grid-cols-3 gap-3">
                <div className="space-y-2">
                  <Label htmlFor="token-limit" className="text-xs">
                    {t('settings.tokenLimit')}
                  </Label>
                  <Input
                    id="token-limit"
                    type="number"
                    value={settings.fixedLimits.tokenLimit}
                    onChange={(e) => handleFixedLimitChange('tokenLimit', e.target.value)}
                    min="1000"
                    max="1000000"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="message-limit" className="text-xs">
                    {t('settings.messageLimit')}
                  </Label>
                  <Input
                    id="message-limit"
                    type="number"
                    value={settings.fixedLimits.messageLimit}
                    onChange={(e) => handleFixedLimitChange('messageLimit', e.target.value)}
                    min="10"
                    max="10000"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="cost-limit" className="text-xs">
                    {t('settings.costLimit')} ($)
                  </Label>
                  <Input
                    id="cost-limit"
                    type="number"
                    step="0.1"
                    value={settings.fixedLimits.costLimit}
                    onChange={(e) => handleFixedLimitChange('costLimit', e.target.value)}
                    min="1"
                    max="1000"
                  />
                </div>
              </div>
            </div>
          )}


          {/* Timezone Selection */}
          <div className="grid grid-cols-4 items-center gap-4">
            <Label htmlFor="timezone" className="text-right">
              {t('settings.timezone')}
            </Label>
            <Select value={settings.timezone} onValueChange={handleTimezoneChange}>
              <SelectTrigger className="col-span-3">
                <SelectValue placeholder={t('settings.timezonePlaceholder')} />
              </SelectTrigger>
              <SelectContent>
                {TIMEZONE_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Refresh Interval */}
          <div className="grid grid-cols-4 items-center gap-4">
            <Label htmlFor="refresh-interval" className="text-right">
              {t('settings.refreshInterval')}
            </Label>
            <Select value={settings.autoRefreshInterval.toString()} onValueChange={handleRefreshIntervalChange}>
              <SelectTrigger className="col-span-3">
                <SelectValue placeholder={t('settings.refreshPlaceholder')} />
              </SelectTrigger>
              <SelectContent>
                {REFRESH_INTERVAL_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value.toString()}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
        <div className="flex justify-end gap-2">
          <Button variant="outline" onClick={handleCancel}>
            {t('settings.cancel')}
          </Button>
          <Button onClick={handleSave} disabled={!hasChanges}>
            {t('settings.save')}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}