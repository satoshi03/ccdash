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
      title: 'Claudeee',
      subtitle: 'Claude Code モニタリング',
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
    session: {
      status: 'ステータス',
      sessionId: 'セッションID',
      startTime: '開始時刻',
      duration: '実行時間',
      tokenUsage: 'トークン使用量',
      messageCount: 'メッセージ数',
      project: 'プロジェクト',
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
    settings: {
      title: '設定',
      description: 'プラン、タイムゾーン、自動更新間隔を設定してください。',
      plan: 'プラン',
      planPlaceholder: 'プランを選択',
      timezone: 'タイムゾーン',
      timezonePlaceholder: 'タイムゾーンを選択',
      refreshInterval: '自動更新間隔',
      refreshPlaceholder: '更新間隔を選択',
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
      title: 'Claudeee',
      subtitle: 'Claude Code Monitoring & Task Scheduler',
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
    session: {
      status: 'Status',
      sessionId: 'Session ID',
      startTime: 'Start Time',
      duration: 'Duration',
      tokenUsage: 'Token Usage',
      messageCount: 'Messages',
      project: 'Project',
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
    settings: {
      title: 'Settings',
      description: 'Configure plan, timezone, and auto-refresh interval.',
      plan: 'Plan',
      planPlaceholder: 'Select plan',
      timezone: 'Timezone',
      timezonePlaceholder: 'Select timezone',
      refreshInterval: 'Auto Refresh Interval',
      refreshPlaceholder: 'Select refresh interval',
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
  let value: any = translations[language]
  
  for (const k of keys) {
    if (value && typeof value === 'object' && k in value) {
      value = value[k]
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
    localStorage.setItem('claudeee-language', language)
  }
}

export function loadLanguage(): Language {
  if (typeof window !== 'undefined') {
    const saved = localStorage.getItem('claudeee-language') as Language
    if (saved && (saved === 'ja' || saved === 'en')) {
      return saved
    }
  }
  // デフォルトは日本語
  return 'ja'
}