
class Component extends DCLogic {
  constructor(props) { super(props); this.state = { lang: props.lang ?? 'fa', copied: false, qr: false, os: 'android', days: 0 }; }
  componentDidMount() { const t0 = performance.now(); const go = (n) => { const t = Math.min(1, (n - t0) / 900); this.setState({ days: 18 * (1 - Math.pow(1 - t, 3)) }); if (t < 1) this.raf = requestAnimationFrame(go); }; this.raf = requestAnimationFrame(go); }
  componentWillUnmount() { cancelAnimationFrame(this.raf); clearTimeout(this.ct); }
  renderVals() {
    const S = this.state, STR = {"en": {"hi": "Hi sara.k", "sub": "Your connection is ready", "left": "days left", "used": "used of", "gb": "GB", "exp": "Renews on 18 December", "copy": "Copy link", "copied": "Link copied", "qr": "Show QR code", "hideqr": "Hide QR code", "connect": "Open in my app", "help": "Set up in 3 steps", "step1": "Install an app", "step2": "Tap “Open in my app”", "step3": "Press connect", "apps": "Pick your phone", "pick": "Recommended", "store": "Get it", "lang": "فارسی", "active": "Active", "need": "Need help? Message your provider", "copiedhint": "Paste it in your app if it did not open by itself", "provider": "Provided by Pars Net"}, "fa": {"hi": "سلام sara.k", "sub": "اتصال شما آماده است", "left": "روز مانده", "used": "مصرف از", "gb": "گیگ", "exp": "تمدید در ۲۷ آذر", "copy": "کپی لینک", "copied": "لینک کپی شد", "qr": "نمایش کد QR", "hideqr": "بستن کد QR", "connect": "باز کردن در برنامه", "help": "راه‌اندازی در ۳ قدم", "step1": "یک برنامه نصب کنید", "step2": "«باز کردن در برنامه» را بزنید", "step3": "دکمه اتصال را بزنید", "apps": "گوشی خود را انتخاب کنید", "pick": "پیشنهادی", "store": "دریافت", "lang": "English", "active": "فعال", "need": "کمک می‌خواهید؟ به فروشنده پیام دهید", "copiedhint": "اگر برنامه خودش باز نشد، لینک را در آن بچسبانید", "provider": "ارائه شده توسط پارس‌نت"}}, s = STR[S.lang];
    const fa = S.lang === 'fa';
    const d = (x) => fa ? String(x).replace(/[0-9]/g, (c) => '۰۱۲۳۴۵۶۷۸۹'[c]).replace('.', '٫') : String(x);
    const APPS = { android: 'v2rayNG', ios: 'Streisand', windows: 'Hiddify', mac: 'Hiddify' };
    const OS = [['android', 'Android'], ['ios', 'iPhone'], ['windows', 'Windows'], ['mac', 'Mac']];
    return { rootCls: 'dl t-' + (this.props.theme ?? 'dark') + ' fx-' + (this.props.effects ?? 'full') + ' ac-' + (this.props.accent ?? 'saffron'), s: s, otherLang: fa ? 'en' : 'fa',
      swapLang: () => this.setState({ lang: fa ? 'en' : 'fa' }), daysS: d(Math.round(S.days)), usedS: d('34'), capS: d('50'), ringOff: (314.2 * (1 - S.days / 30)).toFixed(1),
      copied: S.copied, notCopied: !S.copied, copyLabel: S.copied ? s.copied : s.copy, copyCls: 'btn lg bigcopy' + (S.copied ? ' done copied' : ''),
      copy: () => { this.setState({ copied: true }); clearTimeout(this.ct); this.ct = setTimeout(() => this.setState({ copied: false }), 3000); },
      qr: S.qr, qrExp: S.qr ? 'true' : 'false', qrLabel: S.qr ? s.hideqr : s.qr, toggleQr: () => this.setState({ qr: !S.qr }), connect: (e) => e.preventDefault(),
      oses: OS.map((o) => ({ label: o[1], cls: 'osbtn' + (S.os === o[0] ? ' on' : ''), p: S.os === o[0] ? 'true' : 'false', pick: () => this.setState({ os: o[0] }) })), app: { name: APPS[S.os] } };
  }
}
