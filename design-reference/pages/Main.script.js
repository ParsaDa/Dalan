
class Component extends DCLogic {
  constructor(props) {
    super(props);
    const N = [{"id": "this", "name": "This server", "cc": "", "acc": "saffron", "st": "ok", "lat": 0, "cpu": 0.38, "mem": 0.52, "disk": 0.47, "online": 1284, "rate": 412.0, "tr": 6.3, "users": 1284, "country": "Iran (main)"}, {"id": "ist", "name": "Istanbul", "cc": "TR", "acc": "rose", "st": "ok", "lat": 38, "cpu": 0.41, "mem": 0.56, "disk": 0.3, "online": 212, "rate": 103.6, "tr": 1.8, "users": 212, "country": "Turkey", "chain": 0, "xr": "Running"}, {"id": "fra", "name": "Frankfurt", "cc": "DE", "acc": "sun", "st": "ok", "lat": 61, "cpu": 0.27, "mem": 0.44, "disk": 0.38, "online": 318, "rate": 135.4, "tr": 2.4, "users": 318, "country": "Germany", "chain": 1, "xr": "Running"}, {"id": "ams", "name": "Amsterdam", "cc": "NL", "acc": "tulip", "st": "warn", "lat": 74, "cpu": 0.91, "mem": 0.72, "disk": 0.45999999999999996, "online": 147, "rate": 84.1, "tr": 0.94, "users": 147, "country": "Netherlands", "chain": 0, "xr": "Running"}, {"id": "par", "name": "Paris", "cc": "FR", "acc": "cobalt", "st": "bad", "lat": 0, "cpu": 0, "mem": 0, "disk": 0.54, "online": 0, "rate": 40.0, "tr": 0.61, "users": 0, "country": "France", "chain": 0, "xr": "Unreachable"}, {"id": "lon", "name": "London", "cc": "GB", "acc": "iris", "st": "ok", "lat": 88, "cpu": 0.33, "mem": 0.39, "disk": 0.62, "online": 96, "rate": 68.8, "tr": 1.1, "users": 96, "country": "United Kingdom", "chain": 0, "xr": "Running"}];
    const live = {};
    N.forEach((n, i) => {
      const s = [];
      let v = n.rate || 10;
      for (let k = 0; k < 24; k++) { v = Math.max(5, v * (0.9 + Math.random() * 0.2)); s.push(v); }
      s[23] = n.rate;
      live[n.id] = { cpu: n.cpu, mem: n.mem, disk: n.disk, online: n.online, rate: n.rate, tr: n.tr, spark: s };
    });
    this.N = N;
    this.state = {
      node: 'this', sw: '', rk: 'a', route: 'dash', th: props.theme ?? 'dark', fx: props.effects ?? 'full',
      simple: false, slim: false, search: false, modal: false, restart: false, rsStep: 0, toast: '',
      q: '', cq: '', filter: 'all', newName: '', ago: 4, xray: true, xbusy: false,
      tun: { ist: true, fra: true, ams: true, par: false }, live: live, shown: JSON.parse(JSON.stringify(live)), copied: ''
    };
  }
  componentDidMount() {
    this.iv = setInterval(() => this.tick(), 5000);
    this.sec = setInterval(() => this.setState({ ago: this.state.ago + 1 }), 1000);
    this.vis = () => { if (document.hidden) { clearInterval(this.iv); } else { clearInterval(this.iv); this.iv = setInterval(() => this.tick(), 5000); } };
    document.addEventListener('visibilitychange', this.vis);
  }
  componentWillUnmount() { clearInterval(this.iv); clearInterval(this.sec); cancelAnimationFrame(this.raf); clearTimeout(this.tt); clearTimeout(this.rt); document.removeEventListener('visibilitychange', this.vis); }
  tick() {
    const live = {};
    Object.keys(this.state.live).forEach((id) => {
      const o = this.state.live[id];
      if (id === 'par') { live[id] = o; return; }
      const j = (x, a, lo, hi) => Math.min(hi, Math.max(lo, x + (Math.random() - 0.5) * a));
      const rate = j(o.rate, o.rate * 0.18, 5, 2000);
      live[id] = { cpu: id === 'ams' ? j(o.cpu, 0.04, 0.86, 0.95) : j(o.cpu, 0.08, 0.08, 0.8), mem: j(o.mem, 0.03, 0.2, 0.84), disk: o.disk,
        online: Math.round(j(o.online, Math.max(4, o.online * 0.03), 0, 99999)), rate: rate, tr: o.tr + rate * 5 / 8 / 1e6, spark: o.spark.slice(1).concat([rate]) };
    });
    this.tween(this.state.shown, live);
    this.setState({ live: live, ago: 0 });
  }
  tween(from, to) {
    cancelAnimationFrame(this.raf);
    const reduce = this.state.fx === 'off' || (window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches);
    if (reduce) { this.setState({ shown: to }); return; }
    const t0 = performance.now(), D = 700;
    const ease = (t) => 1 - Math.pow(1 - t, 3);
    const step = (now) => {
      const t = Math.min(1, (now - t0) / D), e = ease(t);
      const shown = {};
      Object.keys(to).forEach((id) => {
        const a = from[id] || to[id], b = to[id];
        shown[id] = { cpu: a.cpu + (b.cpu - a.cpu) * e, mem: a.mem + (b.mem - a.mem) * e, disk: b.disk, online: a.online + (b.online - a.online) * e,
          rate: a.rate + (b.rate - a.rate) * e, tr: a.tr + (b.tr - a.tr) * e, spark: b.spark };
      });
      this.setState({ shown: shown });
      if (t < 1) this.raf = requestAnimationFrame(step);
    };
    this.raf = requestAnimationFrame(step);
  }
  pick(id) {
    if (id === this.state.node) return;
    this.setState({ node: id, sw: this.state.sw === 'a' ? 'b' : 'a', route: this.state.route === 'other' ? 'dash' : this.state.route });
  }
  go(route) { this.setState({ route: route, rk: this.state.rk === 'a' ? 'b' : 'a', search: false }); }
  showToast(msg) { clearTimeout(this.tt); this.setState({ toast: msg }); this.tt = setTimeout(() => this.setState({ toast: '' }), 2600); }
  runRestart() {
    clearTimeout(this.rt);
    this.setState({ restart: true, rsStep: 0 });
    const adv = (k) => { this.rt = setTimeout(() => { if (k < 4) { this.setState({ rsStep: k + 1 }); adv(k + 1); } else { this.setState({ restart: false }); this.showToast('Xray restarted · all users reconnected'); } }, k === 3 ? 900 : 1100); };
    adv(0);
  }
  renderVals() {
    const S = this.state, N = this.N;
    const cur = N.find((n) => n.id === S.node);
    const acc = cur.acc;
    const L = S.shown[cur.id];
    const pct = (x) => Math.round(x * 100) + '%';
    const fmtN = (x) => Math.round(x).toLocaleString('en-US');
    const flags = (cc) => { const o = {}; ['TR','DE','NL','FR','GB','FI','US','IR'].forEach((c) => { o['f_' + c] = c === cc; }); return o; };
    const stf = (st) => ({ s_ok: st === 'ok', s_warn: st === 'warn', s_bad: st === 'bad', s_off: st === 'off' });
    const STW = { ok: 'Online', warn: 'High load', bad: 'Offline', off: 'Paused' };
    const sp = L.spark, W = 300, Hh = 64, pad = 5;
    const mx = Math.max.apply(null, sp), mn = Math.min.apply(null, sp), rg = (mx - mn) || 1;
    const pts = sp.map((v, i) => [pad + i * (W - pad * 2) / (sp.length - 1), pad + (Hh - pad * 2) * (1 - (v - mn) / rg)]);
    const line = 'M' + pts.map((p) => p[0].toFixed(1) + ' ' + p[1].toFixed(1)).join(' L');
    const area = line + ' L' + pts[pts.length - 1][0].toFixed(1) + ' ' + Hh + ' L' + pts[0][0].toFixed(1) + ' ' + Hh + ' Z';
    const v = Object.assign({
      cpu: L.cpu.toFixed(3), mem: L.mem.toFixed(3), disk: L.disk.toFixed(3), cpuP: pct(L.cpu), memP: pct(L.mem), diskP: pct(L.disk),
      cpuBar: 'bar' + (L.cpu >= 0.85 ? ' warn' : ''), onlineS: fmtN(L.online), usersS: fmtN(cur.users), rateS: L.rate.toFixed(1),
      trS: L.tr.toFixed(2) + ' TB', trQ: Math.min(1, L.tr / 10).toFixed(3), line: line, area: area, hx: pts[pts.length - 1][0].toFixed(1), hy: pts[pts.length - 1][1].toFixed(1),
      stw: STW[cur.st], pillCls: 'pill ' + cur.st, offline: cur.st === 'bad', online: cur.st !== 'bad', hasChain: !!cur.chain,
      latS: cur.lat ? cur.lat + ' ms from this server' : 'not answering', country: cur.country
    }, stf(cur.st));
    const go = (r) => () => this.go(r);
    const ICONS = { Dashboard: 'dash', Home: 'dash', Activity: 'activity', Inbounds: 'inbound', Connections: 'inbound', Clients: 'users', Users: 'users', Groups: 'groups', Hosts: 'hosts',
      Tunnels: 'tunnel', Outbounds: 'outbound', Routing: 'routing', Nodes: 'nodes', Servers: 'nodes', 'Tor and PIA': 'tor', 'Xray config': 'xray', Logs: 'logs', Settings: 'settings' };
    const ADV = [['Overview', ['Dashboard', 'Activity']], ['Users and access', ['Inbounds', 'Clients', 'Groups', 'Hosts']], ['Network', ['Tunnels', 'Outbounds', 'Routing', 'Nodes', 'Tor and PIA']], ['System', ['Xray config', 'Logs', 'Settings']]];
    const SIM = [['', ['Home', 'Users', 'Connections', 'Servers', 'Tunnels']], ['', ['Activity', 'Settings']]];
    const routeOf = (lab) => (lab === 'Dashboard' || lab === 'Home') ? 'dash' : (lab === 'Clients' || lab === 'Users') ? 'clients' : 'other:' + lab;
    const curRoute = S.route === 'other' ? 'other:' + S.otherLabel : S.route;
    const navGroups = (S.simple ? SIM : ADV).map((g, gi) => ({ title: g[0], gap: !g[0] && gi > 0, items: g[1].map((lab) => {
      const r = routeOf(lab), on = r === curRoute;
      const o = { label: lab, cls: 'nav' + (on ? ' on' : ''), cur: on ? 'page' : 'false', cnt: (lab === 'Clients' || lab === 'Users') ? '1,284' : '',
        go: () => { if (r.indexOf('other:') === 0) { this.setState({ route: 'other', otherLabel: lab, rk: S.rk === 'a' ? 'b' : 'a' }); } else { this.go(r); } } };
      Object.keys(ICONS).forEach((k) => { o['i_' + ICONS[k]] = o['i_' + ICONS[k]] || ICONS[lab] === ICONS[k]; });
      return o; }) }));
    const tabs = N.map((n) => Object.assign({ name: n.name, cls: 'ntab' + (n.id === S.node ? ' on' : ''), pressed: n.id === S.node ? 'true' : 'false',
      pick: () => this.pick(n.id), isThis: n.id === 'this', lat: n.id === 'this' ? '' : (n.lat ? n.lat + ' ms' : 'offline'), sd: 'sd ' + n.st, stw: STW[n.st] }, flags(n.cc)));
    const cards = N.filter((n) => n.id !== 'this').map((n) => { const l = S.shown[n.id];
      return Object.assign({ name: n.name, pick: () => this.pick(n.id), cls: 'tile edge' + (n.st === 'bad' ? ' st-bad' : n.st === 'warn' ? ' st-warn' : '') + ' ac-' + n.acc + ' t-' + S.th,
        latS: n.lat ? n.lat + ' ms latency' : 'last seen 6 min ago', pillCls: 'pill ' + n.st, stw: STW[n.st], cpu: l.cpu.toFixed(3), mem: l.mem.toFixed(3),
        cpuP: n.st === 'bad' ? '–' : pct(l.cpu), memP: n.st === 'bad' ? '–' : pct(l.mem), cpuBar: 'bar' + (l.cpu >= 0.85 ? ' warn' : ''), tr: n.st === 'bad' ? '610 GB' : (l.tr >= 1 ? l.tr.toFixed(2) + ' TB' : Math.round(l.tr * 1000) + ' GB'),
        xr: n.xr, chain: n.chain ? '1 node behind it · Helsinki' : '' }, flags(n.cc), stf(n.st)); });
    const TUN = [['ist', 'Iran → Istanbul', 'TR', 'BackPack', 'reverse · 38 ms'], ['fra', 'Iran → Frankfurt', 'DE', 'Rakhsh', 'layer 3 · 61 ms'], ['ams', 'Iran → Amsterdam', 'NL', 'BackPack', 'direct · 74 ms'], ['par', 'Iran → Paris', 'FR', 'Rakhsh', 'reverse · no answer']];
    const tunnels = TUN.map((t) => { const on = S.tun[t[0]]; const st = !on ? 'off' : t[0] === 'par' ? 'bad' : 'ok';
      return Object.assign({ name: t[1], engine: t[3], mode: t[4], on: on ? 'true' : 'false', aria: t[1] + ' tunnel', pillCls: 'pill ' + st, word: st === 'off' ? 'Paused' : st === 'bad' ? 'Down' : 'Up',
        flip: () => { const tun = Object.assign({}, S.tun); tun[t[0]] = !on; this.setState({ tun: tun }); this.showToast(t[1] + (on ? ' paused' : ' started')); } }, flags(t[2]), stf(st)); });
    const CL = [['sara.k', 'Family plan', 'ok', 'VLESS', '443', 34, 50, 'Dec 18', true], ['amir_r', 'Gaming', 'ok', 'VLESS', '443', 71, 100, 'Jan 02', true], ['nika.m', 'Family plan', 'warn', 'Trojan', '8443', 47, 50, 'Nov 23', false],
      ['reza.h', 'Office', 'ok', 'VMess', '2053', 12, 200, 'Feb 11', true], ['mina', 'Gaming', 'bad', 'VLESS', '443', 100, 100, 'Expired', false], ['hossein', 'Office', 'ok', 'VLESS', '443', 8, 0, 'Never', true],
      ['shirin.a', 'Family plan', 'off', 'Trojan', '8443', 3, 50, 'Paused', false], ['dariush', '', 'ok', 'VLESS', '443', 63, 100, 'Jan 20', false]];
    const W2 = { ok: 'Active', warn: 'Expiring', bad: 'Expired', off: 'Paused' };
    const cq = S.cq.trim().toLowerCase();
    const counts = { all: CL.length, ok: CL.filter((c) => c[2] === 'ok').length, warn: CL.filter((c) => c[2] === 'warn').length, bad: CL.filter((c) => c[2] === 'bad').length, on: CL.filter((c) => c[8]).length };
    const filt = CL.filter((c) => (S.filter === 'all' || (S.filter === 'on' ? c[8] : c[2] === S.filter)) && (!cq || c[0].indexOf(cq) >= 0));
    const rows = filt.map((c) => Object.assign({ name: c[0], ini: c[0].slice(0, 2).toUpperCase(), group: c[1] || 'No group', word: W2[c[2]], pillCls: 'pill ' + c[2], proto: c[3], port: 'port ' + c[4],
      used: c[5] + ' GB', cap: c[6] ? c[6] + ' GB' : 'Unlimited', q: c[6] ? Math.min(1, c[5] / c[6]).toFixed(2) : '0.08', barCls: 'bar' + (c[6] && c[5] / c[6] >= 0.9 ? ' warn' : ''), exp: c[7],
      on: c[8] ? 'Online' : 'Offline', dot: 'dot sd ' + (c[8] ? 'ok' : 'off'), copyCls: 'ib sm' + (S.copied === c[0] ? ' copied' : ''),
      copy: () => { this.setState({ copied: c[0] }); this.showToast('Subscription link for ' + c[0] + ' copied'); } }, stf(c[2])));
    const filters = [['all', 'All'], ['ok', 'Active'], ['warn', 'Expiring'], ['bad', 'Expired'], ['on', 'Online now']].map((f) => ({ label: f[1], n: counts[f[0]], cls: 'chip' + (S.filter === f[0] ? ' on' : ''), pressed: S.filter === f[0] ? 'true' : 'false', pick: () => this.setState({ filter: f[0] }) }));
    const q = S.q.trim().toLowerCase();
    const RES = [['Users', [['sara.k', 'on Frankfurt · Active', 'users', () => { this.pick('fra'); this.go('clients'); }], ['amir_r', 'on This server · Gaming', 'users', () => { this.pick('this'); this.go('clients'); }]]],
      ['Nodes', N.filter((n) => n.id !== 'this').map((n) => [n.name, STW[n.st] + (n.lat ? ' · ' + n.lat + ' ms' : ''), 'nodes', () => { this.pick(n.id); this.go('dash'); }])],
      ['Inbounds', [['VLESS Reality · 443', 'This server · 812 clients', 'inbound', () => this.setState({ route: 'other', otherLabel: 'Inbounds', search: false, rk: S.rk === 'a' ? 'b' : 'a' })]]],
      ['Actions', [['Add client', 'on the current server', 'plus', () => this.setState({ search: false, modal: true, route: 'clients' })], ['Restart Xray', 'on the current server', 'refresh', () => { this.setState({ search: false }); this.runRestart(); }], ['Effects settings', 'Settings › Appearance', 'settings', () => this.setState({ route: 'other', otherLabel: 'Settings', search: false, rk: S.rk === 'a' ? 'b' : 'a' })]]]];
    const results = RES.map((g) => ({ title: g[0], items: g[1].filter((r) => !q || (r[0] + ' ' + r[1]).toLowerCase().indexOf(q) >= 0).map((r) => {
      const o = { label: r[0], hint: r[1], go: r[3] }; ['users', 'nodes', 'inbound', 'settings', 'refresh', 'plus'].forEach((k) => { o['i_' + k] = k === r[2]; }); return o; }) })).filter((g) => g.items.length);
    const RS = ['Checking the new config', 'Stopping Xray', 'Starting Xray', 'Checking users can connect'];
    const rsSteps = RS.map((l, i) => ({ label: l, cls: 'pstep ' + (i < S.rsStep ? 'done' : i === S.rsStep ? 'now' : 'todo'), done: i < S.rsStep, t: i < S.rsStep ? 'done' : i === S.rsStep ? 'now' : '' }));
    const hour = new Date().getHours();
    const greet = (hour < 5 || hour >= 18 ? 'Good evening' : hour < 12 ? 'Good morning' : 'Good afternoon') + ', Parsa';
    const tunUp = Object.keys(S.tun).filter((k) => S.tun[k] && k !== 'par').length;
    return {
      rootCls: 'dl t-' + S.th + ' fx-' + S.fx + ' ac-' + acc, greet: greet, viewName: cur.name,
      statusWord: cur.st === 'ok' ? 'all systems healthy' : cur.st === 'warn' ? 'running under high load' : 'not answering',
      statusCls: cur.st === 'ok' ? '' : cur.st === 'warn' ? 'pill warn' : 'pill bad', ago: S.ago,
      railCls: 'rail' + (S.slim ? ' slim' : ''), railLabel: S.slim ? 'Expand menu' : 'Collapse menu', railChev: S.slim ? 'scaleX(-1)' : 'none',
      toggleRail: () => this.setState({ slim: !S.slim }), navGroups: navGroups,
      mSimple: S.simple ? 'on' : '', mAdv: S.simple ? '' : 'on', setSimple: () => this.setState({ simple: true }), setAdv: () => this.setState({ simple: false }),
      fxFull: S.fx === 'full' ? 'on' : '', fxLight: S.fx === 'light' ? 'on' : '', fxOff: S.fx === 'off' ? 'on' : '',
      fxSetFull: () => this.setState({ fx: 'full' }), fxSetLight: () => this.setState({ fx: 'light' }), fxSetOff: () => this.setState({ fx: 'off' }),
      isDark: S.th === 'dark', isLight: S.th === 'light', themeLabel: S.th === 'dark' ? 'Switch to light theme' : 'Switch to dark theme',
      toggleTheme: () => this.setState({ th: S.th === 'dark' ? 'light' : 'dark' }),
      tabs: tabs, isThis: S.node === 'this', isNode: S.node !== 'this', v: v, cards: cards, tunnels: tunnels, tunUp: tunUp,
      swCls: S.sw ? 'sw-' + S.sw : 'stag', sweepCls: 'sweep' + (S.sw ? ' go-' + S.sw : ''),
      xrayOn: S.xray ? 'true' : 'false', xrayTg: 'tg' + (S.xbusy ? ' busy' : ''), xrayWord: S.xbusy ? (S.xray ? 'Starting…' : 'Stopping…') : S.xray ? 'Running · uptime 23 d' : 'Stopped · users cannot connect',
      toggleXray: () => { this.setState({ xbusy: true }); setTimeout(() => { this.setState({ xbusy: false, xray: !this.state.xray }); this.showToast(this.state.xray ? 'Xray started' : 'Xray stopped'); }, 1200); },
      restartXray: () => this.runRestart(), rsSteps: rsSteps, rsP: (S.rsStep / 4).toFixed(2), restart: S.restart,
      goParis: () => this.pick('par'), goAms: () => this.pick('ams'), goThis: () => this.pick('this'), goClients: go('clients'), goDash: go('dash'),
      reconnect: () => this.showToast('Asked Paris to reconnect · we will tell you when it answers'),
      rDash: S.route === 'dash', rClients: S.route === 'clients', rOther: S.route === 'other', routeLabel: S.otherLabel || '', rk: S.rk,
      filters: filters, rows: rows, noRows: rows.length === 0, cq: S.cq, onCq: (e) => this.setState({ cq: e.target.value }), clearCq: () => this.setState({ cq: '', filter: 'all' }),
      search: S.search, openSearch: () => this.setState({ search: true, q: '' }), q: S.q, onQ: (e) => this.setState({ q: e.target.value }), results: results, noResults: results.length === 0,
      modal: S.modal, openModal: () => this.setState({ modal: true, newName: '' }), newName: S.newName, onNewName: (e) => this.setState({ newName: e.target.value }),
      saveClient: (e) => { e.preventDefault(); const n = S.newName || 'new client'; this.setState({ modal: false }); this.showToast(n + ' created · link copied'); },
      closeAll: () => this.setState({ search: false, modal: false, restart: false }), stop: (e) => e.stopPropagation(),
      toast: S.toast
    };
  }
}
