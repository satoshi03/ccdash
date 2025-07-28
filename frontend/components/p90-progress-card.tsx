import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Progress } from "@/components/ui/progress"
import { Badge } from "@/components/ui/badge"
import { TrendingUp, Clock, AlertTriangle, Zap } from "lucide-react"
import { useI18n } from "@/hooks/use-i18n"
import { P90Prediction } from "@/lib/api"
import { Settings } from "@/lib/settings"

interface P90ProgressCardProps {
  currentTokens: number
  currentMessages: number
  currentCost: number
  p90Prediction: P90Prediction | null
  plan: string
  resetTime: Date
  isLoading?: boolean
  settings: Settings
}

export function P90ProgressCard({ 
  currentTokens, 
  currentMessages, 
  currentCost, 
  p90Prediction, 
  plan,
  resetTime,
  isLoading = false,
  settings
}: P90ProgressCardProps) {
  const { t, formatFullDate } = useI18n()

  // Determine which limits to use based on settings
  const isUsingFixedLimits = settings.usageMode === 'fixed_limits'
  const tokenLimit = isUsingFixedLimits ? settings.fixedLimits.tokenLimit : p90Prediction?.token_limit || 0
  const messageLimit = isUsingFixedLimits ? settings.fixedLimits.messageLimit : p90Prediction?.message_limit || 0
  const costLimit = isUsingFixedLimits ? settings.fixedLimits.costLimit : p90Prediction?.cost_limit || 0

  if (isLoading || (!isUsingFixedLimits && !p90Prediction)) {
    return (
      <Card className="bg-gradient-to-br from-blue-50 to-indigo-50 border-blue-200">
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <TrendingUp className="w-4 h-4 text-blue-600" />
            {isUsingFixedLimits ? t('settings.fixedLimits') : t('p90Prediction.title')}
          </CardTitle>
          <Badge variant="outline" className="bg-white">
            {plan}
          </Badge>
        </CardHeader>
        <CardContent>
          <div className="animate-pulse space-y-4">
            <div className="space-y-2">
              <div className="h-4 bg-blue-200 rounded w-3/4"></div>
              <div className="h-2 bg-blue-200 rounded"></div>
            </div>
            <div className="space-y-2">
              <div className="h-4 bg-blue-200 rounded w-2/3"></div>
              <div className="h-2 bg-blue-200 rounded"></div>
            </div>
            <div className="space-y-2">
              <div className="h-4 bg-blue-200 rounded w-1/2"></div>
              <div className="h-2 bg-blue-200 rounded"></div>
            </div>
          </div>
        </CardContent>
      </Card>
    )
  }

  const tokenPercentage = tokenLimit > 0 ? (currentTokens / tokenLimit) * 100 : 0
  const messagePercentage = messageLimit > 0 ? (currentMessages / messageLimit) * 100 : 0
  const costPercentage = costLimit > 0 ? (currentCost / costLimit) * 100 : 0

  const isTokenNearLimit = tokenPercentage > 80
  const isMessageNearLimit = messagePercentage > 80
  const isCostNearLimit = costPercentage > 80

  const isAnyNearLimit = isTokenNearLimit || isMessageNearLimit || isCostNearLimit

  const formatTimeToLimit = (minutes: number) => {
    if (minutes <= 0) return t('p90Prediction.alreadyReached')
    if (minutes < 60) return `${minutes}m`
    const hours = Math.floor(minutes / 60)
    const remainingMinutes = minutes % 60
    return `${hours}h ${remainingMinutes}m`
  }

  const formatTimeToReset = (resetTime: Date) => {
    const timeUntilReset = Math.max(0, resetTime.getTime() - Date.now())
    const hoursUntilReset = Math.floor(timeUntilReset / (1000 * 60 * 60))
    const minutesUntilReset = Math.floor((timeUntilReset % (1000 * 60 * 60)) / (1000 * 60))
    
    if (hoursUntilReset === 0 && minutesUntilReset === 0) {
      return t('p90Prediction.resetSoon')
    }
    if (hoursUntilReset === 0) {
      return `${minutesUntilReset}m`
    }
    return `${hoursUntilReset}h ${minutesUntilReset}m`
  }

  const getProgressColor = (percentage: number) => {
    if (percentage > 90) return "bg-red-500"
    if (percentage > 80) return "bg-orange-500"
    if (percentage > 60) return "bg-yellow-500"
    return "bg-green-500"
  }

  return (
    <Card className={`transition-colors ${isAnyNearLimit ? "bg-gradient-to-br from-orange-50 to-red-50 border-orange-200" : "bg-gradient-to-br from-blue-50 to-indigo-50 border-blue-200"}`}>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium flex items-center gap-2">
          <TrendingUp className={`w-4 h-4 ${isAnyNearLimit ? "text-orange-600" : "text-blue-600"}`} />
          {isUsingFixedLimits ? t('settings.fixedLimits') : t('p90Prediction.title')}
        </CardTitle>
        <div className="flex items-center gap-2">
          {!isUsingFixedLimits && p90Prediction && (
            <Badge variant="outline" className="bg-white text-xs">
              {(p90Prediction.confidence * 100).toFixed(0)}% {t('p90Prediction.confidence')}
            </Badge>
          )}
          <Badge variant="outline" className="bg-white">
            {plan}
          </Badge>
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          
          {/* Token Usage */}
          <div className="space-y-2">
            <div className="flex items-center justify-between text-sm">
              <span className="font-medium flex items-center gap-1">
                <Zap className="w-3 h-3" />
                {t('tokenUsage.tokens')}
              </span>
              <div className="flex items-center gap-2">
                <span className="text-muted-foreground">
                  {currentTokens.toLocaleString()} / {tokenLimit.toLocaleString()}
                </span>
                {isTokenNearLimit && <AlertTriangle className="w-3 h-3 text-orange-600" />}
              </div>
            </div>
            <div className="relative">
              <Progress 
                value={Math.min(tokenPercentage, 100)} 
                className="h-2"
              />
              <div 
                className={`absolute top-0 h-2 rounded-full ${getProgressColor(tokenPercentage)} transition-all duration-300`}
                style={{ width: `${Math.min(tokenPercentage, 100)}%` }}
              />
            </div>
            <div className="text-xs text-muted-foreground">
              {tokenPercentage.toFixed(1)}% {isUsingFixedLimits ? 'of fixed limit' : t('p90Prediction.ofPredictedLimit')}
            </div>
          </div>

          {/* Message Usage */}
          <div className="space-y-2">
            <div className="flex items-center justify-between text-sm">
              <span className="font-medium">{t('session.messages')}</span>
              <div className="flex items-center gap-2">
                <span className="text-muted-foreground">
                  {currentMessages} / {messageLimit}
                </span>
                {isMessageNearLimit && <AlertTriangle className="w-3 h-3 text-orange-600" />}
              </div>
            </div>
            <div className="relative">
              <Progress 
                value={Math.min(messagePercentage, 100)} 
                className="h-2"
              />
              <div 
                className={`absolute top-0 h-2 rounded-full ${getProgressColor(messagePercentage)} transition-all duration-300`}
                style={{ width: `${Math.min(messagePercentage, 100)}%` }}
              />
            </div>
            <div className="text-xs text-muted-foreground">
              {messagePercentage.toFixed(1)}% {isUsingFixedLimits ? 'of fixed limit' : t('p90Prediction.ofPredictedLimit')}
            </div>
          </div>

          {/* Cost Usage */}
          {costLimit > 0 && (
            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="font-medium">{t('session.cost')}</span>
                <div className="flex items-center gap-2">
                  <span className="text-muted-foreground">
                    ${currentCost.toFixed(4)} / ${costLimit.toFixed(4)}
                  </span>
                  {isCostNearLimit && <AlertTriangle className="w-3 h-3 text-orange-600" />}
                </div>
              </div>
              <div className="relative">
                <Progress 
                  value={Math.min(costPercentage, 100)} 
                  className="h-2"
                />
                <div 
                  className={`absolute top-0 h-2 rounded-full ${getProgressColor(costPercentage)} transition-all duration-300`}
                  style={{ width: `${Math.min(costPercentage, 100)}%` }}
                />
              </div>
              <div className="text-xs text-muted-foreground">
                {costPercentage.toFixed(1)}% {isUsingFixedLimits ? 'of fixed limit' : t('p90Prediction.ofPredictedLimit')}
              </div>
            </div>
          )}

          {/* Burn Rate and Time Information - Only shown for P90 predictions */}
          {!isUsingFixedLimits && p90Prediction && (
            <div className="border-t pt-3 mt-3">
              <div className={`grid gap-4 text-sm ${p90Prediction.time_to_limit_minutes > 0 ? 'grid-cols-3' : 'grid-cols-2'}`}>
                <div>
                  <div className="text-muted-foreground">{t('p90Prediction.burnRate')}</div>
                  <div className="font-semibold">
                    {p90Prediction.burn_rate_per_hour.toFixed(0)} {t('p90Prediction.tokensPerHour')}
                  </div>
                </div>
                {p90Prediction.time_to_limit_minutes > 0 && (
                  <div>
                    <div className="text-muted-foreground flex items-center gap-1">
                      <Clock className="w-3 h-3" />
                      {t('p90Prediction.timeToLimit')}
                    </div>
                    <div className={`font-semibold ${p90Prediction.time_to_limit_minutes < 60 ? "text-red-600" : p90Prediction.time_to_limit_minutes < 180 ? "text-orange-600" : ""}`}>
                      {formatTimeToLimit(p90Prediction.time_to_limit_minutes)}
                    </div>
                  </div>
                )}
                <div>
                  <div className="text-muted-foreground flex items-center gap-1">
                    <Clock className="w-3 h-3" />
                    {t('p90Prediction.resetIn')}
                  </div>
                  <div className="font-semibold text-blue-600" title={formatFullDate(resetTime)}>
                    {formatTimeToReset(resetTime)}
                  </div>
                  <div className="text-xs text-muted-foreground mt-1">
                    {t('p90Prediction.at')} {resetTime.toLocaleTimeString([], { 
                      hour: '2-digit', 
                      minute: '2-digit',
                      timeZoneName: 'short'
                    })}
                  </div>
                </div>
              </div>

              {/* Last Updated */}
              <div className="text-xs text-muted-foreground border-t pt-2 col-span-full">
                {t('p90Prediction.lastPredicted')}: {formatFullDate(new Date(p90Prediction.predicted_at))}
              </div>
            </div>
          )}

          {/* Reset Time - Only shown for fixed limits mode */}
          {isUsingFixedLimits && (
            <div className="border-t pt-3 mt-3">
              <div className="grid gap-4 text-sm grid-cols-1">
                <div>
                  <div className="text-muted-foreground flex items-center gap-1">
                    <Clock className="w-3 h-3" />
                    {t('p90Prediction.resetIn')}
                  </div>
                  <div className="font-semibold text-blue-600" title={formatFullDate(resetTime)}>
                    {formatTimeToReset(resetTime)}
                  </div>
                  <div className="text-xs text-muted-foreground mt-1">
                    {t('p90Prediction.at')} {resetTime.toLocaleTimeString([], { 
                      hour: '2-digit', 
                      minute: '2-digit',
                      timeZoneName: 'short'
                    })}
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  )
}