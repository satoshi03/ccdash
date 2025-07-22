import { redirect } from 'next/navigation'

export default function EnglishHomePage() {
  // 英語版の場合は共通のページコンポーネントを使用
  redirect('/?locale=en')
}