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
import { Settings, getSettings, saveSettings, TIMEZONE_OPTIONS, REFRESH_INTERVAL_OPTIONS } from "@/lib/settings"
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

  const handlePlanChange = (value: string) => {
    const newSettings = { ...settings, plan: value as Settings['plan'] }
    setSettings(newSettings)
    setHasChanges(true)
  }

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
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>{t('settings.title')}</DialogTitle>
          <DialogDescription>
            {t('settings.description')}
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          <div className="grid grid-cols-4 items-center gap-4">
            <Label htmlFor="plan" className="text-right">
              {t('settings.plan')}
            </Label>
            <Select value={settings.plan} onValueChange={handlePlanChange}>
              <SelectTrigger className="col-span-3">
                <SelectValue placeholder={t('settings.planPlaceholder')} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="Pro">Claude Pro (7,000トークン)</SelectItem>
                <SelectItem value="Max5">Claude Max5 (35,000トークン)</SelectItem>
                <SelectItem value="Max20">Claude Max20 (140,000トークン)</SelectItem>
              </SelectContent>
            </Select>
          </div>
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