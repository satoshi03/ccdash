export type Language = 'ja' | 'en'

export interface Translations {
  [key: string]: string | Translations
}

// 翻訳辞書
export const translations: Record<Language, Translations> = {
  ja: {
    common: {
      refresh: '更新',
      back: '戻る',
      loading: '読み込み中...',
      error: 'エラー',
      copy: 'コピー',
      copied: 'コピー済み',
      settings: '設定',
      overview: '概要',
      sessions: 'セッション',
    },
    header: {
      title: 'CCDash',
      subtitle: 'Claude Code ダッシュボード',
      nav: {
        dashboard: 'ダッシュボード',
        dashboardDesc: 'トークン使用量とプロジェクト概要',
        sessions: 'セッション',
        sessionsDesc: 'Claude Codeセッションの詳細',
        scheduler: 'スケジューラー',
        schedulerDesc: 'タスクの自動スケジューリング',
        resources: 'リソース',
        claudeCodeDesc: 'Claude Codeの公式ドキュメント',
        github: 'GitHub',
        githubDesc: 'ソースコードを確認',
        issues: '課題',
        issuesDesc: 'バグ報告や機能要求',
        documentation: 'ドキュメント',
        documentationDesc: '使い方とガイド',
      },
    },
    tokenUsage: {
      title: 'トークン使用量',
      available: '使用可能',
      tokens: 'トークン',
      estimate: '（見積もり）',
      excess: '超過分',
      usageRate: '使用率',
      resetIn: 'リセットまで',
      hours: '時間',
      minutes: '分',
    },
    p90Prediction: {
      title: '消費量(P90予測)',
      tokenLimit: 'トークンリミット',
      messageLimit: 'メッセージリミット',
      costLimit: 'コストリミット',
      confidence: '信頼度',
      burnRate: '消費率',
      timeToLimit: '予測到達時間',
      resetIn: 'リセットまで',
      tokensPerHour: 'トークン/時',
      ofPredictedLimit: '予測値',
      alreadyReached: '超過',
      resetSoon: 'リセット予定',
      lastPredicted: '最終予測',
      at: '時刻',
    },
    session: {
      status: 'ステータス',
      sessionId: 'セッションID',
      startTime: '開始時刻',
      duration: '実行時間',
      tokenUsage: 'トークン使用量',
      messageCount: 'メッセージ数',
      project: 'プロジェクト',
      cost: 'コスト',
      messages: 'メッセージ',
      active: '実行中',
      completed: '完了',
      paused: '停止中',
      failed: '失敗',
      unknown: '不明',
      detail: 'セッション詳細',
      info: 'セッション情報',
      lastActivity: '最終活動',
      messageHistory: 'メッセージ履歴',
      user: 'ユーザー',
      assistant: 'アシスタント',
      codeGenerated: '生成されたコード',
      codeBlocks: 'コードブロック',
      input: '入力',
      output: '出力',
      total: '合計',
      thinking: '内部思考',
      toolResult: 'Tool Result',
      page: 'ページ',
      noContent: 'No content',
      rawObjectContent: 'Raw object content:',
      unknownItemType: 'Unknown item type:',
    },
    errors: {
      tokenUsageFetch: 'トークン使用量の取得に失敗しました',
      sessionsFetch: 'セッション情報の取得に失敗しました',
      sessionNotFound: 'セッションが見つかりません',
      sessionDetailFetch: 'セッション詳細を読み込み中...',
    },
    empty: {
      noSessions: 'セッションがありません',
      noSessionsDesc: 'Claude Codeを使用してセッションが作成されると、ここに表示されます',
      noProjects: 'プロジェクトがありません',
      noProjectsDesc: 'Claude Codeを使用してプロジェクトが作成されると、ここに表示されます',
    },
    project: {
      totalTokenUsage: '総トークン使用量',
      sessionCount: 'セッション数',
      recentSessions: '最近のセッション',
      running: '実行中',
      completed: '完了',
      others: '他',
      sessionsText: 'セッション',
      messages: 'メッセージ',
      tokensLabel: 'tokens',
    },
    job: {
      // Basic job terms
      title: 'ジョブ詳細',
      history: 'ジョブ履歴',
      execution: 'タスク実行',
      command: 'コマンド',
      output: '出力',
      error: 'エラー',
      logs: 'ログ',
      
      // Status
      status: {
        running: '実行中',
        completed: '完了',
        pending: '待機中',
        failed: '失敗',
        cancelled: 'キャンセル済み',
      },
      
      // Schedule types
      schedule: {
        immediate: '即時実行',
        afterReset: 'リセット後',
        delayed: '遅延実行',
        scheduled: '時刻指定',
      },
      
      // Time related
      duration: '実行時間',
      executionTime: '実行時間',
      completedAt: '完了日時',
      scheduledAt: '予定実行時刻',
      timeUntil: '実行まで',
      hoursAfter: '時間後',
      minutesAfter: '分後',
      daysAfter: '日後',
      
      // Actions and messages
      create: 'ジョブを作成',
      cancel: 'キャンセル',
      delete: '削除',
      executing: '実行中...',
      created: 'ジョブが正常に作成されました',
      notFound: 'ジョブが見つかりません',
      loading: 'ジョブ詳細を読み込み中...',
      confirmDelete: 'このジョブを削除しますか？',
      
      // Output logs
      outputLog: '出力ログ',
      errorLog: 'エラーログ',
      standardOutput: '標準出力',
      standardError: '標準エラー出力',
      noOutputLog: '出力ログがありません',
      noErrorLog: 'エラーログがありません',
      
      // Form validation
      validation: {
        futureTime: 'スケジュール日時は現在時刻より後に設定してください',
        withinYear: 'スケジュール日時は1年以内に設定してください',
      },
      
      // Descriptions
      description: 'Claude Codeタスクを実行します。プロジェクトを選択してコマンドを入力してください。',
      resetDescription: 'セッションウィンドウがリセットされた後に実行されます。',
      delayDescription: '時間後に実行されます。',
      scheduleDescription: 'に実行されます。',
      
      // Counters
      totalJobs: '件のジョブを表示',
      
      // Table headers
      scheduleHeader: 'スケジュール',
      createdAt: '作成日時',
      actions: 'アクション',
      
      // Task execution form
      form: {
        selectProject: 'プロジェクトを選択してください',
        loadingProjects: '読み込み中...',
        errorProjects: 'エラー',
        noProjects: 'プロジェクトがありません',
        executionDirectory: '実行ディレクトリ',
        commandPlaceholder: '例: 新しい機能を実装して...',
        commandHelp: 'Claude Codeに実行させたいタスクを自然言語で記述してください。',
        yoloMode: 'YOLOモード',
        yoloModeDescription: '(確認なしで変更を実行)',
        executionTiming: '実行タイミング',
        executionDate: '実行日',
        executionTime: '実行時刻',
        executionScheduled: '実行予定',
        dateTimeRequired: '日付と時刻を両方指定してください',
      },
      
      // Security warnings
      security: {
        warningTitle: 'セキュリティ警告',
        yoloWarning: 'YOLOモードは全てのコマンドを安全性チェックなしで実行します。悪意のあるコマンドによってシステムが損傷する可能性があります。',
        safeExecutionTitle: '安全な実行のために:',
        safeExecutionText: 'Claude Codeの設定ファイルでコマンドの許可を適切に行ってください。',
        settingsGuide: '設定方法を確認する →',
      },
    },
    settings: {
      title: '設定',
      description: 'タイムゾーン、自動更新間隔、使用量表示モードを設定してください。',
      timezone: 'タイムゾーン',
      timezonePlaceholder: 'タイムゾーンを選択',
      refreshInterval: '自動更新間隔',
      refreshPlaceholder: '更新間隔を選択',
      usageMode: '使用量表示モード',
      usageModePlaceholder: '表示モードを選択',
      p90Prediction: 'P90予測',
      fixedLimits: '固定値',
      fixedLimitsConfig: '固定値設定',
      preset: 'プリセット',
      tokenLimit: 'トークン制限',
      messageLimit: 'メッセージ制限',
      costLimit: 'コスト制限',
      cancel: 'キャンセル',
      save: '保存',
    },
    footer: {
      description: 'Claude Code使用状況の監視とタスクスケジューリングを行うWebアプリケーション',
      sponsor: 'スポンサー',
      sections: {
        product: 'プロダクト',
        resources: 'リソース',
        support: 'サポート',
      },
      links: {
        dashboard: 'ダッシュボード',
        sessions: 'セッション',
        scheduler: 'スケジューラー',
        documentation: 'ドキュメント',
        claudeCode: 'Claude Code',
        github: 'GitHub',
        issues: '課題・バグ報告',
        discussions: 'ディスカッション',
        contribute: '貢献する',
        privacy: 'プライバシー',
        terms: '利用規約',
      },
      rights: 'All rights reserved.',
      madeWith: '作成者:',
      forCommunity: 'コミュニティのために',
      builtWith: '技術スタック',
      status: 'ステータス',
      statusOperational: '正常稼働中',
    },
  },
  en: {
    common: {
      refresh: 'Refresh',
      back: 'Back',
      loading: 'Loading...',
      error: 'Error',
      copy: 'Copy',
      copied: 'Copied',
      settings: 'Settings',
      overview: 'Overview',
      sessions: 'Sessions',
    },
    header: {
      title: 'CCDash',
      subtitle: 'Claude Code Dashboard',
      nav: {
        dashboard: 'Dashboard',
        dashboardDesc: 'Token usage and project overview',
        sessions: 'Sessions',
        sessionsDesc: 'Claude Code session details',
        scheduler: 'Scheduler',
        schedulerDesc: 'Automatic task scheduling',
        resources: 'Resources',
        claudeCodeDesc: 'Official Claude Code documentation',
        github: 'GitHub',
        githubDesc: 'View source code',
        issues: 'Issues',
        issuesDesc: 'Bug reports and feature requests',
        documentation: 'Documentation',
        documentationDesc: 'Usage guides and documentation',
      },
    },
    tokenUsage: {
      title: 'Token Usage',
      available: 'Available',
      tokens: 'tokens',
      estimate: '(estimate)',
      excess: 'Excess',
      usageRate: 'Usage Rate',
      resetIn: 'Reset in',
      hours: 'hours',
      minutes: 'minutes',
    },
    p90Prediction: {
      title: 'P90 Usage Predictions',
      tokenLimit: 'Token Prediction Limit',
      messageLimit: 'Message Prediction Limit',
      costLimit: 'Cost Prediction Limit',
      confidence: 'Confidence',
      burnRate: 'Burn Rate',
      timeToLimit: 'Time to Limit',
      resetIn: 'Reset In',
      tokensPerHour: 'tokens/hour',
      ofPredictedLimit: 'of predicted limit',
      alreadyReached: 'Already reached',
      resetSoon: 'Reset soon',
      lastPredicted: 'Last predicted',
      at: 'at',
    },
    session: {
      status: 'Status',
      sessionId: 'Session ID',
      startTime: 'Start Time',
      duration: 'Duration',
      tokenUsage: 'Token Usage',
      messageCount: 'Messages',
      project: 'Project',
      cost: 'Cost',
      messages: 'Messages',
      active: 'Active',
      completed: 'Completed',
      paused: 'Paused',
      failed: 'Failed',
      unknown: 'Unknown',
      detail: 'Session Detail',
      info: 'Session Information',
      lastActivity: 'Last Activity',
      messageHistory: 'Message History',
      user: 'User',
      assistant: 'Assistant',
      codeGenerated: 'Generated Code',
      codeBlocks: 'code blocks',
      input: 'Input',
      output: 'Output',
      total: 'Total',
      thinking: 'Internal Thinking',
      toolResult: 'Tool Result',
      page: 'Page',
      noContent: 'No content',
      rawObjectContent: 'Raw object content:',
      unknownItemType: 'Unknown item type:',
    },
    errors: {
      tokenUsageFetch: 'Failed to fetch token usage',
      sessionsFetch: 'Failed to fetch session information',
      sessionNotFound: 'Session not found',
      sessionDetailFetch: 'Loading session details...',
    },
    empty: {
      noSessions: 'No sessions',
      noSessionsDesc: 'Sessions will appear here when you use Claude Code',
      noProjects: 'No projects',
      noProjectsDesc: 'Projects will appear here when you use Claude Code',
    },
    project: {
      totalTokenUsage: 'Total Token Usage',
      sessionCount: 'Sessions',
      recentSessions: 'Recent Sessions',
      running: 'Running',
      completed: 'Completed',
      others: 'others',
      sessionsText: 'sessions',
      messages: 'messages',
      tokensLabel: 'tokens',
    },
    job: {
      // Basic job terms
      title: 'Job Details',
      history: 'Job History',
      execution: 'Task Execution',
      command: 'Command',
      output: 'Output',
      error: 'Error',
      logs: 'Logs',
      
      // Status
      status: {
        running: 'Running',
        completed: 'Completed',
        pending: 'Pending',
        failed: 'Failed',
        cancelled: 'Cancelled',
      },
      
      // Schedule types
      schedule: {
        immediate: 'Immediate',
        afterReset: 'After Reset',
        delayed: 'Delayed',
        scheduled: 'Scheduled',
      },
      
      // Time related
      duration: 'Duration',
      executionTime: 'Execution Time',
      completedAt: 'Completed At',
      scheduledAt: 'Scheduled At',
      timeUntil: 'Time Until',
      hoursAfter: 'hours later',
      minutesAfter: 'minutes later',
      daysAfter: 'days later',
      
      // Actions and messages
      create: 'Create Job',
      cancel: 'Cancel',
      delete: 'Delete',
      executing: 'Executing...',
      created: 'Job created successfully',
      notFound: 'Job not found',
      loading: 'Loading job details...',
      confirmDelete: 'Are you sure you want to delete this job?',
      
      // Output logs
      outputLog: 'Output Log',
      errorLog: 'Error Log',
      standardOutput: 'Standard Output',
      standardError: 'Standard Error',
      noOutputLog: 'No output log available',
      noErrorLog: 'No error log available',
      
      // Form validation
      validation: {
        futureTime: 'Schedule time must be set in the future',
        withinYear: 'Schedule time must be within one year',
      },
      
      // Descriptions
      description: 'Execute Claude Code tasks. Select a project and enter a command.',
      resetDescription: 'Will be executed after the session window is reset.',
      delayDescription: 'Will be executed after the specified hours.',
      scheduleDescription: ' will be executed.',
      
      // Counters
      totalJobs: 'jobs displayed',
      
      // Table headers
      scheduleHeader: 'Schedule',
      createdAt: 'Created At',
      actions: 'Actions',
      
      // Task execution form
      form: {
        selectProject: 'Select a project',
        loadingProjects: 'Loading...',
        errorProjects: 'Error',
        noProjects: 'No projects available',
        executionDirectory: 'Execution Directory',
        commandPlaceholder: 'e.g., Implement a new feature...',
        commandHelp: 'Describe the task you want Claude Code to execute in natural language.',
        yoloMode: 'YOLO Mode',
        yoloModeDescription: '(Execute changes without confirmation)',
        executionTiming: 'Execution Timing',
        executionDate: 'Execution Date',
        executionTime: 'Execution Time',
        executionScheduled: 'Scheduled for',
        dateTimeRequired: 'Please specify both date and time',
      },
      
      // Security warnings
      security: {
        warningTitle: 'Security Warning',
        yoloWarning: 'YOLO mode executes all commands without safety checks. Malicious commands could damage your system.',
        safeExecutionTitle: 'For safe execution:',
        safeExecutionText: 'Configure command permissions appropriately in the Claude Code settings file.',
        settingsGuide: 'View configuration guide →',
      },
    },
    settings: {
      title: 'Settings',
      description: 'Configure timezone, auto-refresh interval, and usage display mode.',
      timezone: 'Timezone',
      timezonePlaceholder: 'Select timezone',
      refreshInterval: 'Auto Refresh Interval',
      refreshPlaceholder: 'Select refresh interval',
      usageMode: 'Usage Display Mode',
      usageModePlaceholder: 'Select display mode',
      p90Prediction: 'P90 Prediction',
      fixedLimits: 'Fixed Limits',
      fixedLimitsConfig: 'Fixed Limits Configuration',
      preset: 'Preset',
      tokenLimit: 'Token Limit',
      messageLimit: 'Message Limit',
      costLimit: 'Cost Limit',
      cancel: 'Cancel',
      save: 'Save',
    },
    footer: {
      description: 'Web application for monitoring Claude Code usage and task scheduling',
      sponsor: 'Sponsor',
      sections: {
        product: 'Product',
        resources: 'Resources',
        support: 'Support',
      },
      links: {
        dashboard: 'Dashboard',
        sessions: 'Sessions',
        scheduler: 'Scheduler',
        documentation: 'Documentation',
        claudeCode: 'Claude Code',
        github: 'GitHub',
        issues: 'Issues & Bug Reports',
        discussions: 'Discussions',
        contribute: 'Contribute',
        privacy: 'Privacy',
        terms: 'Terms',
      },
      rights: 'All rights reserved.',
      madeWith: 'Made with',
      forCommunity: 'for the community',
      builtWith: 'Built with',
      status: 'Status',
      statusOperational: 'Operational',
    },
  },
}

// 翻訳取得関数
export function getTranslation(language: Language, key: string): string {
  const keys = key.split('.')
  let value: unknown = translations[language]
  
  for (const k of keys) {
    if (value && typeof value === 'object' && k in value) {
      value = value[k as keyof typeof value]
    } else {
      // フォールバック: 日本語を試す
      if (language !== 'ja') {
        return getTranslation('ja', key)
      }
      return key // キーが見つからない場合はキー自体を返す
    }
  }
  
  return typeof value === 'string' ? value : key
}

// 言語に応じたロケール設定
export function getLocale(language: Language): string {
  switch (language) {
    case 'ja':
      return 'ja-JP'
    case 'en':
      return 'en-US'
    default:
      return 'ja-JP'
  }
}

// 日時フォーマット関数
export function formatDateTime(language: Language, date: Date): string {
  const locale = getLocale(language)
  return date.toLocaleString(locale, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  })
}

export function formatFullDateTime(language: Language, date: Date): string {
  const locale = getLocale(language)
  return date.toLocaleString(locale, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

// 言語設定の保存/読み込み
export function saveLanguage(language: Language): void {
  if (typeof window !== 'undefined') {
    localStorage.setItem('ccdash-language', language)
  }
}

export function loadLanguage(): Language {
  if (typeof window !== 'undefined') {
    const saved = localStorage.getItem('ccdash-language') as Language
    if (saved && (saved === 'ja' || saved === 'en')) {
      return saved
    }
  }
  // デフォルトは日本語
  return 'ja'
}