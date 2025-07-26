import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Progress } from "@/components/ui/progress"
import { Badge } from "@/components/ui/badge"
import { Clock, AlertTriangle } from "lucide-react"
import { useI18n } from "@/hooks/use-i18n"

interface TokenUsageCardProps {
  currentUsage: number
  usageLimit: number
  plan: string
  resetTime: Date
  availableTokens?: number
  totalCost?: number
  totalMessages?: number
}

export function TokenUsageCard({ currentUsage, usageLimit, plan, resetTime, availableTokens, totalCost, totalMessages }: TokenUsageCardProps) {
  const { t, formatFullDate } = useI18n()
  const usagePercentage = (currentUsage / usageLimit) * 100
  const isNearLimit = usagePercentage > 80
  const isOverLimit = currentUsage > usageLimit
  const timeUntilReset = Math.max(0, resetTime.getTime() - Date.now())
  const hoursUntilReset = Math.floor(timeUntilReset / (1000 * 60 * 60))
  const minutesUntilReset = Math.floor((timeUntilReset % (1000 * 60 * 60)) / (1000 * 60))
  
  const remainingTokens = usageLimit - currentUsage
  const availableCount = availableTokens ?? Math.max(0, remainingTokens)
  const excessTokens = isOverLimit ? currentUsage - usageLimit : 0
  
  const formatResetTime = (date: Date) => {
    return formatFullDate(date)
  }

  return (
    <Card className={isOverLimit ? "border-red-200 bg-red-50" : isNearLimit ? "border-orange-200 bg-orange-50" : ""}>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">{t('tokenUsage.title')}</CardTitle>
        <Badge variant={isOverLimit ? "destructive" : isNearLimit ? "destructive" : "secondary"}>{plan}</Badge>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <div className="text-2xl font-bold">{currentUsage.toLocaleString()}</div>
            <div className="text-sm text-muted-foreground">/ {usageLimit.toLocaleString()} {t('tokenUsage.tokens')}</div>
          </div>
          
          {totalCost !== undefined && totalCost > 0 && (
            <div className="flex items-center justify-between">
              <div className="text-lg font-semibold text-green-600">${totalCost.toFixed(4)}</div>
              <div className="text-sm text-muted-foreground">USD</div>
            </div>
          )}
          
          {totalMessages !== undefined && totalMessages > 0 && (
            <div className="flex items-center justify-between">
              <div className="text-lg font-semibold text-blue-600">{totalMessages.toLocaleString()}</div>
              <div className="text-sm text-muted-foreground">messages (current window)</div>
            </div>
          )}
          
          <div className="flex items-center justify-between text-sm text-muted-foreground">
            <span>{t('tokenUsage.available')}: {availableCount.toLocaleString()}{t('tokenUsage.tokens')}</span>
            <span className="text-xs">{t('tokenUsage.estimate')}</span>
          </div>

          {isOverLimit && (
            <div className="flex items-center justify-between text-sm">
              <span className="text-red-600 font-medium">
                {t('tokenUsage.excess')}: {excessTokens.toLocaleString()}{t('tokenUsage.tokens')}
              </span>
              <AlertTriangle className="w-4 h-4 text-red-600" />
            </div>
          )}

          <Progress value={usagePercentage} className={`w-full ${isNearLimit ? "bg-orange-100" : ""}`} />

          <div className="flex items-center justify-between text-sm">
            <div className="flex items-center gap-2">
              <span className={isOverLimit ? "text-red-600" : isNearLimit ? "text-orange-600" : "text-muted-foreground"}>
                {t('tokenUsage.usageRate')}: {usagePercentage.toFixed(1)}%
              </span>
              {(isNearLimit || isOverLimit) && !isOverLimit && <AlertTriangle className="w-4 h-4 text-orange-600" />}
            </div>

            <div className="flex items-center gap-1 text-muted-foreground">
              <Clock className="w-4 h-4" />
              <span title={formatResetTime(resetTime)}>
                {t('tokenUsage.resetIn')} {hoursUntilReset}{t('tokenUsage.hours')}{minutesUntilReset}{t('tokenUsage.minutes')}
              </span>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
