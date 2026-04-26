import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import zhCN from './locales/zh-CN.json'
import enUS from './locales/en-US.json'

const resources = {
  'zh-CN': { translation: zhCN },
  'en-US': { translation: enUS },
}

function getLanguage(): string {
  // Priority: localStorage.lang > navigator.language > fallback
  const stored = localStorage.getItem('lang')
  if (stored && Object.keys(resources).includes(stored)) {
    return stored
  }

  const browserLang = navigator.language
  if (Object.keys(resources).includes(browserLang)) {
    return browserLang
  }

  // Check if browserLang is a variant (e.g., "en" for "en-US")
  const langPrefix = browserLang.split('-')[0]
  const matched = Object.keys(resources).find((k) => k.startsWith(langPrefix))
  if (matched) {
    return matched
  }

  return 'zh-CN'
}

i18n.use(initReactI18next).init({
  resources,
  lng: getLanguage(),
  fallbackLng: 'zh-CN',
  interpolation: {
    escapeValue: false,
  },
})

export default i18n

// Simple translation hook
export function useTran() {
  return i18n.t.bind(i18n)
}

// Hook for changing language
export function useChangeLanguage() {
  return (lang: string) => {
    i18n.changeLanguage(lang)
    localStorage.setItem('lang', lang)
  }
}

// Export current language
export function getCurrentLanguage(): string {
  return i18n.language
}
