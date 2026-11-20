
class Component extends DCLogic {
  constructor(p) { super(p); this.state = { k: 0 }; }
  renderVals() { return { rootCls: 'dl t-' + (this.props.theme ?? 'dark') + ' fx-' + (this.props.effects ?? 'full') + ' ac-' + (this.props.accent ?? 'saffron'), mkA: this.state.k % 2 === 0, mkB: this.state.k % 2 === 1, replay: () => this.setState({ k: this.state.k + 1 }) }; }
}
