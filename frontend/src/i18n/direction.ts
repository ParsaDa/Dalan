const RTL_LANGUAGES = new Set(['fa-IR', 'ar-EG']);

export const isRtlLanguage = (language: string): boolean => RTL_LANGUAGES.has(language);
