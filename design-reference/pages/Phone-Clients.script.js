
class Component extends DCLogic {
  renderVals() {
    const th = this.props.theme ?? 'dark';
    const fx = this.props.effects ?? 'full';
    const ac = this.props.accent ?? 'saffron';
    return { rootCls: 'dl t-' + th + ' fx-' + fx + ' ac-' + ac + ' phone ' };
  }
}
