
class Component extends DCLogic {
  constructor(p) { super(p); this.state = { n: 0, sw: '', live: 1284, shown: 1284, sp: [3,4,4,6,5,7,8,7,9,10,9,11], st: 'ok', flash: false, tg: true, cp: false, chip: 0, save: 0, m: false, ek: 0, r: 0, rk: 'a' }; }
  componentWillUnmount() { cancelAnimationFrame(this.raf); clearTimeout(this.t1); clearTimeout(this.t2); }
  count(from, to) { cancelAnimationFrame(this.raf); const t0 = performance.now(); const f = (n) => { const t = Math.min(1, (n - t0) / 700); this.setState({ shown: from + (to - from) * (1 - Math.pow(1 - t, 3)) }); if (t < 1) this.raf = requestAnimationFrame(f); }; this.raf = requestAnimationFrame(f); }
  renderVals() {
    const S = this.state, NN = [['This server', 'saffron', '38%', '1,284'], ['Istanbul', 'rose', '41%', '212'], ['Frankfurt', 'sun', '27%', '318'], ['London', 'iris', '33%', '96']];
    const cur = NN[S.n % NN.length];
    const sp = S.sp, W = 300, Hh = 50, mx = Math.max.apply(null, sp), mn = Math.min.apply(null, sp);
    const pts = sp.map((v, i) => [4 + i * (W - 8) / (sp.length - 1), 4 + (Hh - 8) * (1 - (v - mn) / ((mx - mn) || 1))]);
    const line = 'M' + pts.map((p) => p[0].toFixed(1) + ' ' + p[1].toFixed(1)).join(' L');
    return { rootCls: 'dl t-' + (this.props.theme ?? 'dark') + ' fx-' + (this.props.effects ?? 'full') + ' ac-' + (this.props.accent ?? 'saffron'),
      miniCls: 'ac-' + cur[1] + ' t-' + (this.props.theme ?? 'dark'), sweepCls: 'sweep' + (S.sw ? ' go-' + S.sw : ''), swCls: S.sw ? 'sw-' + S.sw : '',
      mini: NN.slice(0, 3).map((x, i) => ({ name: x[0], cls: 'ntab' + (x[0] === cur[0] ? ' on' : '') })), cpuS: cur[2], onS: cur[3],
      nextNode: () => this.setState({ n: S.n + 1, sw: S.sw === 'a' ? 'b' : 'a' }),
      liveS: Math.round(S.shown).toLocaleString('en-US'), liveBar: Math.min(1, S.shown / 2000).toFixed(3), line: line, area: line + ' L' + pts[pts.length - 1][0].toFixed(1) + ' ' + Hh + ' L4 ' + Hh + ' Z',
      hx: pts[pts.length - 1][0].toFixed(1), hy: pts[pts.length - 1][1].toFixed(1), pillCls: 'pill ' + S.st + (S.flash ? ' flash' : ''), pillW: S.st === 'ok' ? 'Online' : 'High load',
      bump: () => { const nv = Math.round(S.live * (0.9 + Math.random() * 0.2)); const st = Math.random() < 0.35 ? (S.st === 'ok' ? 'warn' : 'ok') : S.st; this.count(S.shown, nv);
        this.setState({ live: nv, sp: sp.slice(1).concat([nv / 120]), st: st, flash: st !== S.st }); clearTimeout(this.t1); this.t1 = setTimeout(() => this.setState({ flash: false }), 900); },
      tgOn: S.tg ? 'true' : 'false', flip: () => this.setState({ tg: !S.tg }),
      cpCls: 'btn sm' + (S.cp ? ' copied' : ''), cpLabel: S.cp ? 'Copied' : 'Copy link', copy: () => { this.setState({ cp: true }); clearTimeout(this.t2); this.t2 = setTimeout(() => this.setState({ cp: false }), 1600); },
      chips: ['All', 'Online', 'Expiring'].map((l, i) => ({ l: l, cls: 'chip' + (S.chip === i ? ' on' : ''), pick: () => this.setState({ chip: i }) })),
      saveL: S.save === 1 ? 'Saving…' : S.save === 2 ? 'Saved' : 'Save', save: () => { this.setState({ save: 1 }); setTimeout(() => this.setState({ save: 2 }), 700); setTimeout(() => this.setState({ save: 0 }), 2200); },
      m: S.m, openM: () => this.setState({ m: true }), closeM: () => this.setState({ m: false }), stop: (e) => e.stopPropagation(),
      entCls: S.ek % 2 ? 'sw-a' : 'stag', replayE: () => this.setState({ ek: S.ek + 1 }),
      routes: ['Dashboard', 'Clients', 'Tunnels'].map((l, i) => ({ l: l, cls: 'chip' + (S.r % 3 === i ? ' on' : '') })), routeName: ['Dashboard', 'Clients', 'Tunnels'][S.r % 3],
      routeCls: 'route-' + S.rk, nextRoute: () => this.setState({ r: S.r + 1, rk: S.rk === 'a' ? 'b' : 'a' }) };
  }
}
