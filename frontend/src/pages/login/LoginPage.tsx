import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { message } from 'antd';

import { useForm } from 'react-hook-form';
import { HttpUtil, LanguageManager } from '@/utils';
import { rhfZodValidate } from '@/components/form/rhf';
import { setMessageInstance } from '@/utils/messageBus';
import { pauseAnimationsUntilLeave, useTheme } from '@/hooks/useTheme';
import Icon from '@/components/dl/Icon';
import Mark from '@/components/dl/Mark';
import { DropChip } from '@/components/dl/DropChip';
import { Sk } from '@/components/dl/primitives';
import { LoginFormSchema, TwoFactorCodeSchema, type LoginFormValues } from '@/schemas/login';

type LoginForm = LoginFormValues;

const basePath = window.X_UI_BASE_PATH || '';

export default function LoginPage() {
  const { t } = useTranslation();
  const { isDark, isUltra, toggleTheme, toggleUltra } = useTheme();
  const [messageApi, messageContextHolder] = message.useMessage();

  useEffect(() => {
    setMessageInstance(messageApi);
  }, [messageApi]);

  const [fetched, setFetched] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [twoFactorEnable, setTwoFactorEnable] = useState(false);
  const [lang, setLang] = useState<string>(() => LanguageManager.getLanguage());

  useEffect(() => {
    let cancelled = false;
    (async () => {
      const msg = await HttpUtil.post('/getTwoFactorEnable');
      if (cancelled) return;
      if (msg.success) setTwoFactorEnable(!!msg.obj);
      setFetched(true);
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const onSubmit = useCallback(async (values: LoginForm) => {
    setSubmitting(true);
    try {
      const msg = await HttpUtil.post('/login', values);
      if (msg.success) window.location.href = basePath + 'panel/';
    } finally {
      setSubmitting(false);
    }
  }, []);

  const onLangChange = useCallback((next: string) => {
    setLang(next);
    LanguageManager.setLanguage(next);
  }, []);

  const cycleTheme = useCallback(() => {
    pauseAnimationsUntilLeave('login-theme-cycle');
    if (!isDark) {
      toggleTheme();
      if (isUltra) toggleUltra();
    } else if (!isUltra) {
      toggleUltra();
    } else {
      toggleUltra();
      toggleTheme();
    }
  }, [isDark, isUltra, toggleTheme, toggleUltra]);

  const [showPassword, setShowPassword] = useState(false);
  const { register, handleSubmit, formState } = useForm<LoginForm>({
    defaultValues: { username: '', password: '', twoFactorCode: '' },
  });
  const fieldError = (name: keyof LoginForm) => {
    const message = formState.errors[name]?.message;
    return message ? <small style={{ color: 'var(--bad)' }}>{t(String(message))}</small> : null;
  };
  const languages: readonly { value: string; name: string }[] = LanguageManager.supportedLanguages;

  return (
    <>
      {messageContextHolder}
      <div className="bgfx" aria-hidden="true">
        <div className="glow g1" />
        <div className="glow g2" />
        <div className="glow g3" />
      </div>
      <main
        style={{
          minHeight: '100vh',
          display: 'grid',
          placeItems: 'center',
          padding: '40px 20px',
          position: 'relative',
        }}
      >
        <div className="col" style={{ alignItems: 'center', gap: 26, width: 'min(400px, 100%)' }}>
          <div className="col" style={{ alignItems: 'center', gap: 12 }}>
            <Mark size={76} motion="anim" title="Dalan" />
            <div className="wm" style={{ fontSize: 32 }}>
              Dalan
            </div>
            <h1 className="mut" style={{ fontWeight: 650, fontSize: 15, margin: 0 }}>
              {t('loginPage.subtitle')}
            </h1>
          </div>
          {!fetched ? (
            <div className="tile" style={{ width: '100%', padding: 24, borderRadius: 24 }}>
              <Sk h={180} />
            </div>
          ) : (
            <form
              className="tile"
              style={{
                width: '100%',
                padding: 24,
                borderRadius: 24,
                display: 'flex',
                flexDirection: 'column',
                gap: 14,
              }}
              onSubmit={handleSubmit(onSubmit)}
              noValidate
            >
              <label className="field">
                <span>{t('username')}</span>
                <input
                  className="inp"
                  autoComplete="username"
                  autoFocus
                  aria-invalid={!!formState.errors.username}
                  {...register('username', {
                    validate: rhfZodValidate(LoginFormSchema.shape.username),
                  })}
                />
                {fieldError('username')}
              </label>
              <label className="field">
                <span>{t('password')}</span>
                <div style={{ position: 'relative' }}>
                  <input
                    className="inp"
                    type={showPassword ? 'text' : 'password'}
                    autoComplete="current-password"
                    aria-invalid={!!formState.errors.password}
                    style={{ width: '100%', paddingInlineEnd: 44 }}
                    {...register('password', {
                      validate: rhfZodValidate(LoginFormSchema.shape.password),
                    })}
                  />
                  <button
                    type="button"
                    className="ib sm"
                    aria-label={
                      showPassword ? t('loginPage.hidePassword') : t('loginPage.showPassword')
                    }
                    aria-pressed={showPassword}
                    onClick={() => setShowPassword(!showPassword)}
                    style={{
                      position: 'absolute',
                      insetInlineEnd: 5,
                      top: 5,
                      background: 'transparent',
                      border: 0,
                    }}
                  >
                    <Icon name="eye" />
                  </button>
                </div>
                {fieldError('password')}
              </label>
              {twoFactorEnable && (
                <label className="field">
                  <span>{t('twoFactorCode')}</span>
                  <input
                    className="inp num"
                    inputMode="numeric"
                    autoComplete="one-time-code"
                    placeholder={t('loginPage.sixDigits')}
                    style={{ letterSpacing: '.3em' }}
                    aria-invalid={!!formState.errors.twoFactorCode}
                    {...register('twoFactorCode', {
                      validate: rhfZodValidate(TwoFactorCodeSchema),
                    })}
                  />
                  {fieldError('twoFactorCode')}
                </label>
              )}
              <button
                type="submit"
                className="btn pri lg full"
                disabled={submitting}
                aria-busy={submitting}
              >
                {t('login')}
              </button>
            </form>
          )}
          <div className="row wrap" style={{ gap: 10, justifyContent: 'center' }}>
            <div className="seg" role="group" aria-label={t('pages.settings.language')}>
              {['en-US', 'fa-IR'].map((code) => (
                <button
                  key={code}
                  type="button"
                  className={lang === code ? 'on' : ''}
                  aria-pressed={lang === code}
                  onClick={() => onLangChange(code)}
                >
                  {code === 'fa-IR' ? 'فارسی' : 'English'}
                </button>
              ))}
            </div>
            <DropChip
              label={t('loginPage.more')}
              value={['en-US', 'fa-IR'].includes(lang) ? '' : lang}
              onChange={(v) => v && onLangChange(v)}
              options={[
                { value: '', label: '…' },
                ...languages
                  .filter((l) => !['en-US', 'fa-IR'].includes(l.value))
                  .map((l) => ({ value: l.value, label: l.name })),
              ]}
            />
            <button
              type="button"
              id="login-theme-cycle"
              className="ib sm"
              aria-label={t('menu.theme')}
              title={t('menu.theme')}
              onClick={cycleTheme}
            >
              <Icon name={isDark ? 'moon' : 'sun'} />
            </button>
          </div>
        </div>
      </main>
    </>
  );
}
