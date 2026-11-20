
class Component extends DCLogic {
  constructor(p) { super(p); this.state = { k: 0, t: 0 }; }
  componentDidMount() { this.start(); }
  componentWillUnmount() { clearInterval(this.iv); }
  start() { clearInterval(this.iv); this.setState({ t: 0 }); this.iv = setInterval(() => { const t = this.state.t + 1; this.setState({ t: t }); if (t >= 14) clearInterval(this.iv); }, 600); }
  renderVals() {
    const t = this.state.t, L = ['Downloading Rakhsh 0.9', 'Checking the file signature', 'Installing the service', 'Opening the tunnel to this server', 'Testing the tunnel'];
    const at = [0, 3, 5, 8, 11, 14];
    const cur = at.findIndex((a, i) => t >= a && t < (at[i + 1] ?? 99));
    const p = Math.min(1, t / 14);
    return { rootCls: 'dl t-' + (this.props.theme ?? 'dark') + ' fx-' + (this.props.effects ?? 'full') + ' ac-' + (this.props.accent ?? 'sun'),
      loading: t < 3, ready: t >= 3, p: p.toFixed(3), pctS: Math.round(p * 100) + '%', replay: () => this.start(),
      steps: L.map((l, i) => ({ label: l, done: i < cur || t >= 14, cls: 'pstep ' + (i < cur || t >= 14 ? 'done' : i === cur ? 'now' : 'todo'), t: i < cur || t >= 14 ? 'done' : i === cur ? 'working' : '' })) };
  }
}
