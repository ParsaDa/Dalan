// Three soft glows behind the glass; two take the current accent, one is mint.
export default function PanelBackground() {
  return (
    <div className="dl-bgfx" aria-hidden>
      <i className="dl-glow g1" />
      <i className="dl-glow g2" />
      <i className="dl-glow g3" />
    </div>
  );
}
