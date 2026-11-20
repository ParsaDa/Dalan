
class Component extends DCLogic {
  constructor(props) { super(props); this.state = { node: 'this', sw: '', more: false, add: false, q: '', th: props.theme ?? 'dark', fx: props.effects ?? 'full', bump: 0 }; this.N = [{"id": "this", "name": "This server", "cc": "", "acc": "saffron", "st": "ok", "lat": 0, "online": 1284, "cpu": 0.38}, {"id": "ist", "name": "Istanbul", "cc": "TR", "acc": "rose", "st": "ok", "lat": 38, "online": 212, "cpu": 0.41}, {"id": "fra", "name": "Frankfurt", "cc": "DE", "acc": "sun", "st": "ok", "lat": 61, "online": 318, "cpu": 0.27}, {"id": "ams", "name": "Amsterdam", "cc": "NL", "acc": "tulip", "st": "warn", "lat": 74, "online": 147, "cpu": 0.91}, {"id": "par", "name": "Paris", "cc": "FR", "acc": "cobalt", "st": "bad", "lat": 0, "online": 0, "cpu": 0}, {"id": "lon", "name": "London", "cc": "GB", "acc": "iris", "st": "ok", "lat": 88, "online": 96, "cpu": 0.33}]; }
  componentDidMount() { this.iv = setInterval(() => this.setState({ bump: this.state.bump + 1 }), 5000); }
  componentWillUnmount() { clearInterval(this.iv); }
  componentDidUpdate(pp) { if (pp.theme !== this.props.theme || pp.effects !== this.props.effects) this.setState({ th: this.props.theme ?? 'dark', fx: this.props.effects ?? 'full' }); }
  pick(id) { if (id !== this.state.node) this.setState({ node: id, sw: this.state.sw === 'a' ? 'b' : 'a' }); }
  renderVals() {
    const S = this.state, N = this.N, cur = N.find((n) => n.id === S.node);
    const STW = { ok: 'Online', warn: 'High load', bad: 'Offline' };
    const fl = (cc) => { const o = {}; ['TR','DE','NL','FR','GB','FI','US','IR'].forEach((c) => { o['f_' + c] = c === cc; }); return o; };
    const stf = (st) => ({ s_ok: st === 'ok', s_warn: st === 'warn', s_bad: st === 'bad', s_off: st === 'off' });
    const mk = (n) => Object.assign({ name: n.name, isThis: n.id === 'this', cls: 'ntab' + (n.id === S.node ? ' on' : ''), pressed: n.id === S.node ? 'true' : 'false', sd: 'sd ' + n.st, stw: STW[n.st], pillCls: 'pill ' + n.st,
      pick: () => this.pick(n.id), pickClose: () => { this.pick(n.id); this.setState({ more: false }); } }, fl(n.cc), stf(n.st));
    const shown = N.slice(0, 3); if (!shown.find((n) => n.id === S.node)) shown[2] = cur;
    const q = S.q.trim().toLowerCase();
    const jit = (x, a) => x + Math.sin(S.bump * 1.7 + x) * a;
    const h = new Date().getHours();
    return { rootCls: 'dl t-' + S.th + ' fx-' + S.fx + ' ac-' + cur.acc, greet: (h < 5 || h >= 18 ? 'Good evening' : h < 12 ? 'Good morning' : 'Good afternoon') + ', Parsa',
      name: cur.name, stWord: cur.st === 'ok' ? 'all healthy' : cur.st === 'warn' ? 'high load' : 'not answering', stCls: cur.st === 'ok' ? 'mut' : 'pill ' + cur.st,
      chips: shown.map(mk), all: N.filter((n) => !q || n.name.toLowerCase().indexOf(q) >= 0).map(mk), moreN: N.length - 3, moreLabel: 'All ' + N.length + ' servers',
      openMore: () => this.setState({ more: true, q: '' }), openAdd: () => this.setState({ add: true }), close: () => this.setState({ more: false, add: false }), q: S.q, onQ: (e) => this.setState({ q: e.target.value }),
      openSearch: () => this.setState({ more: true, q: '' }), more: S.more, add: S.add,
      swCls: S.sw ? 'sw-' + S.sw : 'stag', sweepCls: 'sweep' + (S.sw ? ' go-' + S.sw : ''), off: cur.st === 'bad', on: cur.st !== 'bad',
      onlineS: Math.round(jit(cur.online, Math.max(3, cur.online * 0.02))).toLocaleString('en-US'), cpu: Math.max(0, Math.min(1, jit(cur.cpu, 0.04))).toFixed(3), cpuP: Math.round(Math.max(0, Math.min(1, jit(cur.cpu, 0.04))) * 100) + '%',
      cpuBar: 'bar' + (cur.cpu >= 0.85 ? ' warn' : ''), pillCls: 'pill ' + cur.st, stw: STW[cur.st], v: stf(cur.st) };
  }
}
