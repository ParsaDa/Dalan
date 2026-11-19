export interface OutboundRow {
  key: number;
  tag?: string;
  protocol?: string;
  streamSettings?: { network?: string; security?: string };
  settings?: Record<string, unknown>;
  // Set on a tunnel's row: it is read-only here and managed on the Tunnels page.
  tunnelEngine?: string;
  // Set on a Tor exit's row: it is read-only here and managed in the Tor dialog.
  torCountry?: string;
}
