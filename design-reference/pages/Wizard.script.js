
class Component extends DCLogic {
  constructor(p) { super(p); this.state = { step: 0, k: 'a', preset: 'reality', srv: 'Tehran main', user: '', copied: false }; }
  go(n) { this.setState({ step: n, k: this.state.k === 'a' ? 'b' : 'a' }); }
  renderVals() {
    const S = this.state, L = ['Server', 'Connection', 'First user'];
    const PRE = [["reality", "Best for most people", "VLESS Reality on port 443. Looks like normal HTTPS, hard to block.", true], ["cdn", "Behind a CDN", "Trojan over WebSocket. Use when direct IPs get blocked.", false], ["game", "Low ping", "VLESS on a layer 3 tunnel. For gaming and calls.", false]];
    return { rootCls: 'dl t-' + (this.props.theme ?? 'dark') + ' fx-' + (this.props.effects ?? 'full') + ' ac-' + (this.props.accent ?? 'saffron'),
      steps: L.map((l, i) => ({ label: l, n: i + 1, done: i < S.step, notDone: i >= S.step, cls: 'st' + (i < S.step ? ' done' : i === S.step ? ' on' : ''), line: i < 2, lnCls: 'ln' + (i < S.step ? ' done' : '') })),
      st0: S.step === 0, st1: S.step === 1, st2: S.step === 2, st3: S.step === 3, stepAnim: 'wz-' + S.k, nav: S.step < 3, noBack: S.step === 0,
      back: () => this.go(Math.max(0, S.step - 1)), next: () => this.go(S.step + 1), nextLabel: S.step === 2 ? 'Create and finish' : 'Continue',
      srvName: S.srv, onSrv: (e) => this.setState({ srv: e.target.value }), user: S.user, onUser: (e) => this.setState({ user: e.target.value }), userOrYou: S.user || 'Your first user',
      presets: PRE.map((p) => ({ title: p[1], desc: p[2], rec: p[3], on: S.preset === p[0] ? 'true' : 'false', cls: 'tile' + (S.preset === p[0] ? ' edge' : ''), pick: () => this.setState({ preset: p[0] }) })),
      copy: () => this.setState({ copied: true }), copyLabel: S.copied ? 'Copied' : 'Copy', copyCls: 'btn pri' + (S.copied ? ' copied' : '') };
  }
}
