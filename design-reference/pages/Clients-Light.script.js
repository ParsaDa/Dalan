
class Component extends DCLogic {
  renderVals() {
    const th = this.props.theme ?? 'light';
    const fx = this.props.effects ?? 'full';
    const ac = this.props.accent ?? 'sun';
    return { rootCls: 'dl t-' + th + ' fx-' + fx + ' ac-' + ac + ' ' };
  }
}
