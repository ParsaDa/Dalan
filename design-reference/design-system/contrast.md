# Colour contrast

Every text and graphic pair in both themes, measured against the real background: plain `bg`, `surface`, glass at Full effects over the glow at its brightest spot (accent at 14% dark / 16% light under 7% / 60% white), and dense glass. Thresholds: text 4.5:1, large text and icons 3:1, lines, bars and chart strokes 3:1. Light-mode shades carry a 0.15 safety margin. 272 pairs, all pass.

Two changes from the brief came out of this check:

- **Muted text, dark:** `#A8A399` measured 4.05:1 over glass on a 20% saffron glow. It is lifted to `#AFAAA0`, and the glow is capped at 14%, which together give 4.5:1 or better everywhere.
- **Warning and danger words, dark:** the status fills stay as in the brief (`#FF7A45`, `#F87171`) for icons, dots and bars (3:1). Words use `#FE8759` and `#FF8482`, lifted at the same hue so they pass 4.5:1 over glass and glow.
- Light saffron: the brief's `#B87D00` / `#8A5E00` both pass on cream; the system uses the computed `#B57A04` / `#906102`, which also pass on glass over a 16% glow.

| Theme | Foreground | Hex | Background | Hex | Ratio | Needs | Use | Pass |
|---|---|---|---|---|---|---|---|---|
| both | on-accent on Saffron | `#111114` | Saffron fill | `#FFB020` | 10.31:1 | 4.5:1 | button/chip label | yes |
| dark | Saffron line | `#FFB020` | bg | `#111114` | 10.31:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Saffron text | `#FFB020` | bg | `#111114` | 10.31:1 | 4.5:1 | active tab label, links | yes |
| dark | Saffron line | `#FFB020` | surface | `#1B1B20` | 9.38:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Saffron text | `#FFB020` | surface | `#1B1B20` | 9.38:1 | 4.5:1 | active tab label, links | yes |
| dark | Saffron line | `#FFB020` | glass (Full) | `#403626` | 6.47:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Saffron text | `#FFB020` | glass (Full) | `#403626` | 6.47:1 | 4.5:1 | active tab label, links | yes |
| dark | Saffron line | `#FFB020` | glass dense | `#1E1C1F` | 9.25:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Saffron text | `#FFB020` | glass dense | `#1E1C1F` | 9.25:1 | 4.5:1 | active tab label, links | yes |
| light | Saffron line | `#B57A04` | bg | `#F4EFE6` | 3.19:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Saffron text | `#906102` | bg | `#F4EFE6` | 4.71:1 | 4.5:1 | active tab label, links | yes |
| light | Saffron line | `#B57A04` | surface | `#FFFFFF` | 3.65:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Saffron text | `#906102` | surface | `#FFFFFF` | 5.39:1 | 4.5:1 | active tab label, links | yes |
| light | Saffron line | `#B57A04` | glass (Full) | `#FBF5E8` | 3.36:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Saffron text | `#906102` | glass (Full) | `#FBF5E8` | 4.96:1 | 4.5:1 | active tab label, links | yes |
| light | Saffron line | `#B57A04` | glass dense | `#FEFCF8` | 3.56:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Saffron text | `#906102` | glass dense | `#FEFCF8` | 5.26:1 | 4.5:1 | active tab label, links | yes |
| both | on-accent on Rose | `#111114` | Rose fill | `#FF3D6E` | 5.52:1 | 4.5:1 | button/chip label | yes |
| dark | Rose line | `#FF3D6E` | bg | `#111114` | 5.52:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Rose text | `#FE6380` | bg | `#111114` | 6.56:1 | 4.5:1 | active tab label, links | yes |
| dark | Rose line | `#FF3D6E` | surface | `#1B1B20` | 5.02:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Rose text | `#FE6380` | surface | `#1B1B20` | 5.97:1 | 4.5:1 | active tab label, links | yes |
| dark | Rose line | `#FF3D6E` | glass (Full) | `#402731` | 3.96:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Rose text | `#FE6380` | glass (Full) | `#402731` | 4.71:1 | 4.5:1 | active tab label, links | yes |
| dark | Rose line | `#FF3D6E` | glass dense | `#1E1B20` | 4.98:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Rose text | `#FE6380` | glass dense | `#1E1B20` | 5.93:1 | 4.5:1 | active tab label, links | yes |
| light | Rose line | `#F93669` | bg | `#F4EFE6` | 3.18:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Rose text | `#D3034F` | bg | `#F4EFE6` | 4.71:1 | 4.5:1 | active tab label, links | yes |
| light | Rose line | `#F93669` | surface | `#FFFFFF` | 3.65:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Rose text | `#D3034F` | surface | `#FFFFFF` | 5.39:1 | 4.5:1 | active tab label, links | yes |
| light | Rose line | `#F93669` | glass (Full) | `#FBEDED` | 3.2:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Rose text | `#D3034F` | glass (Full) | `#FBEDED` | 4.73:1 | 4.5:1 | active tab label, links | yes |
| light | Rose line | `#F93669` | glass dense | `#FEFAFA` | 3.52:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Rose text | `#D3034F` | glass dense | `#FEFAFA` | 5.2:1 | 4.5:1 | active tab label, links | yes |
| both | on-accent on Sun | `#111114` | Sun fill | `#FFC83D` | 12.19:1 | 4.5:1 | button/chip label | yes |
| dark | Sun line | `#FFC83D` | bg | `#111114` | 12.19:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Sun text | `#FFC83D` | bg | `#111114` | 12.19:1 | 4.5:1 | active tab label, links | yes |
| dark | Sun line | `#FFC83D` | surface | `#1B1B20` | 11.09:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Sun text | `#FFC83D` | surface | `#1B1B20` | 11.09:1 | 4.5:1 | active tab label, links | yes |
| dark | Sun line | `#FFC83D` | glass (Full) | `#403A2A` | 7.31:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Sun text | `#FFC83D` | glass (Full) | `#403A2A` | 7.31:1 | 4.5:1 | active tab label, links | yes |
| dark | Sun line | `#FFC83D` | glass dense | `#1E1D1F` | 10.86:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Sun text | `#FFC83D` | glass dense | `#1E1D1F` | 10.86:1 | 4.5:1 | active tab label, links | yes |
| light | Sun line | `#A98006` | bg | `#F4EFE6` | 3.17:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Sun text | `#866505` | bg | `#F4EFE6` | 4.73:1 | 4.5:1 | active tab label, links | yes |
| light | Sun line | `#A98006` | surface | `#FFFFFF` | 3.63:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Sun text | `#866505` | surface | `#FFFFFF` | 5.42:1 | 4.5:1 | active tab label, links | yes |
| light | Sun line | `#A98006` | glass (Full) | `#FBF6EA` | 3.37:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Sun text | `#866505` | glass (Full) | `#FBF6EA` | 5.02:1 | 4.5:1 | active tab label, links | yes |
| light | Sun line | `#A98006` | glass dense | `#FEFCF9` | 3.55:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Sun text | `#866505` | glass dense | `#FEFCF9` | 5.29:1 | 4.5:1 | active tab label, links | yes |
| both | on-accent on Tulip | `#111114` | Tulip fill | `#F062C8` | 6.54:1 | 4.5:1 | button/chip label | yes |
| dark | Tulip line | `#F062C8` | bg | `#111114` | 6.54:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Tulip text | `#F466CC` | bg | `#111114` | 6.84:1 | 4.5:1 | active tab label, links | yes |
| dark | Tulip line | `#F062C8` | surface | `#1B1B20` | 5.95:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Tulip text | `#F466CC` | surface | `#1B1B20` | 6.22:1 | 4.5:1 | active tab label, links | yes |
| dark | Tulip line | `#F062C8` | glass (Full) | `#3E2C3C` | 4.47:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Tulip text | `#F466CC` | glass (Full) | `#3E2C3C` | 4.67:1 | 4.5:1 | active tab label, links | yes |
| dark | Tulip line | `#F062C8` | glass dense | `#1E1B22` | 5.9:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Tulip text | `#F466CC` | glass dense | `#1E1B22` | 6.17:1 | 4.5:1 | active tab label, links | yes |
| light | Tulip line | `#DB4EB5` | bg | `#F4EFE6` | 3.18:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Tulip text | `#BB2D98` | bg | `#F4EFE6` | 4.65:1 | 4.5:1 | active tab label, links | yes |
| light | Tulip line | `#DB4EB5` | surface | `#FFFFFF` | 3.64:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Tulip text | `#BB2D98` | surface | `#FFFFFF` | 5.33:1 | 4.5:1 | active tab label, links | yes |
| light | Tulip line | `#DB4EB5` | glass (Full) | `#FAEFF3` | 3.24:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Tulip text | `#BB2D98` | glass (Full) | `#FAEFF3` | 4.75:1 | 4.5:1 | active tab label, links | yes |
| light | Tulip line | `#DB4EB5` | glass dense | `#FEFAFB` | 3.52:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Tulip text | `#BB2D98` | glass dense | `#FEFAFB` | 5.15:1 | 4.5:1 | active tab label, links | yes |
| both | on-accent on Cobalt | `#111114` | Cobalt fill | `#4C8DFF` | 5.89:1 | 4.5:1 | button/chip label | yes |
| dark | Cobalt line | `#4C8DFF` | bg | `#111114` | 5.89:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Cobalt text | `#619AFF` | bg | `#111114` | 6.8:1 | 4.5:1 | active tab label, links | yes |
| dark | Cobalt line | `#4C8DFF` | surface | `#1B1B20` | 5.36:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Cobalt text | `#619AFF` | surface | `#1B1B20` | 6.19:1 | 4.5:1 | active tab label, links | yes |
| dark | Cobalt line | `#4C8DFF` | glass (Full) | `#293143` | 4.06:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Cobalt text | `#619AFF` | glass (Full) | `#293143` | 4.69:1 | 4.5:1 | active tab label, links | yes |
| dark | Cobalt line | `#4C8DFF` | glass dense | `#1B1C23` | 5.3:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Cobalt text | `#619AFF` | glass dense | `#1B1C23` | 6.12:1 | 4.5:1 | active tab label, links | yes |
| light | Cobalt line | `#4283F4` | bg | `#F4EFE6` | 3.17:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Cobalt text | `#2766D4` | bg | `#F4EFE6` | 4.66:1 | 4.5:1 | active tab label, links | yes |
| light | Cobalt line | `#4283F4` | surface | `#FFFFFF` | 3.63:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Cobalt text | `#2766D4` | surface | `#FFFFFF` | 5.33:1 | 4.5:1 | active tab label, links | yes |
| light | Cobalt line | `#4283F4` | glass (Full) | `#F0F2F7` | 3.24:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Cobalt text | `#2766D4` | glass (Full) | `#F0F2F7` | 4.76:1 | 4.5:1 | active tab label, links | yes |
| light | Cobalt line | `#4283F4` | glass dense | `#FAFBFC` | 3.5:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Cobalt text | `#2766D4` | glass dense | `#FAFBFC` | 5.15:1 | 4.5:1 | active tab label, links | yes |
| both | on-accent on Iris | `#111114` | Iris fill | `#9B8CFF` | 6.81:1 | 4.5:1 | button/chip label | yes |
| dark | Iris line | `#9B8CFF` | bg | `#111114` | 6.81:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Iris text | `#9D90FE` | bg | `#111114` | 7.05:1 | 4.5:1 | active tab label, links | yes |
| dark | Iris line | `#9B8CFF` | surface | `#1B1B20` | 6.2:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Iris text | `#9D90FE` | surface | `#1B1B20` | 6.42:1 | 4.5:1 | active tab label, links | yes |
| dark | Iris line | `#9B8CFF` | glass (Full) | `#333143` | 4.57:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Iris text | `#9D90FE` | glass (Full) | `#333143` | 4.73:1 | 4.5:1 | active tab label, links | yes |
| dark | Iris line | `#9B8CFF` | glass dense | `#1C1C23` | 6.12:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Iris text | `#9D90FE` | glass dense | `#1C1C23` | 6.34:1 | 4.5:1 | active tab label, links | yes |
| light | Iris line | `#8675E6` | bg | `#F4EFE6` | 3.22:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Iris text | `#6C59C7` | bg | `#F4EFE6` | 4.71:1 | 4.5:1 | active tab label, links | yes |
| light | Iris line | `#8675E6` | surface | `#FFFFFF` | 3.68:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Iris text | `#6C59C7` | surface | `#FFFFFF` | 5.4:1 | 4.5:1 | active tab label, links | yes |
| light | Iris line | `#8675E6` | glass (Full) | `#F5F2F7` | 3.32:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Iris text | `#6C59C7` | glass (Full) | `#F5F2F7` | 4.86:1 | 4.5:1 | active tab label, links | yes |
| light | Iris line | `#8675E6` | glass dense | `#FCFBFC` | 3.57:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Iris text | `#6C59C7` | glass dense | `#FCFBFC` | 5.23:1 | 4.5:1 | active tab label, links | yes |
| both | on-accent on Lime | `#111114` | Lime fill | `#B6F23C` | 14.14:1 | 4.5:1 | button/chip label | yes |
| dark | Lime line | `#B6F23C` | bg | `#111114` | 14.14:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Lime text | `#B6F23C` | bg | `#111114` | 14.14:1 | 4.5:1 | active tab label, links | yes |
| dark | Lime line | `#B6F23C` | surface | `#1B1B20` | 12.87:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Lime text | `#B6F23C` | surface | `#1B1B20` | 12.87:1 | 4.5:1 | active tab label, links | yes |
| dark | Lime line | `#B6F23C` | glass (Full) | `#373E2A` | 8.36:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Lime text | `#B6F23C` | glass (Full) | `#373E2A` | 8.36:1 | 4.5:1 | active tab label, links | yes |
| dark | Lime line | `#B6F23C` | glass dense | `#1D1E1F` | 12.53:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Lime text | `#B6F23C` | glass dense | `#1D1E1F` | 12.53:1 | 4.5:1 | active tab label, links | yes |
| light | Lime line | `#6A9206` | bg | `#F4EFE6` | 3.2:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Lime text | `#547500` | bg | `#F4EFE6` | 4.68:1 | 4.5:1 | active tab label, links | yes |
| light | Lime line | `#6A9206` | surface | `#FFFFFF` | 3.67:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Lime text | `#547500` | surface | `#FFFFFF` | 5.36:1 | 4.5:1 | active tab label, links | yes |
| light | Lime line | `#6A9206` | glass (Full) | `#F7F9EA` | 3.44:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Lime text | `#547500` | glass (Full) | `#F7F9EA` | 5.02:1 | 4.5:1 | active tab label, links | yes |
| light | Lime line | `#6A9206` | glass dense | `#FCFDF9` | 3.59:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Lime text | `#547500` | glass dense | `#FCFDF9` | 5.24:1 | 4.5:1 | active tab label, links | yes |
| both | on-accent on Cyan | `#111114` | Cyan fill | `#2CCCE4` | 9.75:1 | 4.5:1 | button/chip label | yes |
| dark | Cyan line | `#2CCCE4` | bg | `#111114` | 9.75:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Cyan text | `#2CCCE4` | bg | `#111114` | 9.75:1 | 4.5:1 | active tab label, links | yes |
| dark | Cyan line | `#2CCCE4` | surface | `#1B1B20` | 8.87:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Cyan text | `#2CCCE4` | surface | `#1B1B20` | 8.87:1 | 4.5:1 | active tab label, links | yes |
| dark | Cyan line | `#2CCCE4` | glass (Full) | `#253A3F` | 6.19:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Cyan text | `#2CCCE4` | glass (Full) | `#253A3F` | 6.19:1 | 4.5:1 | active tab label, links | yes |
| dark | Cyan line | `#2CCCE4` | glass dense | `#1A1D22` | 8.74:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Cyan text | `#2CCCE4` | glass dense | `#1A1D22` | 8.74:1 | 4.5:1 | active tab label, links | yes |
| light | Cyan line | `#0494A6` | bg | `#F4EFE6` | 3.17:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Cyan text | `#037685` | bg | `#F4EFE6` | 4.66:1 | 4.5:1 | active tab label, links | yes |
| light | Cyan line | `#0494A6` | surface | `#FFFFFF` | 3.63:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Cyan text | `#037685` | surface | `#FFFFFF` | 5.34:1 | 4.5:1 | active tab label, links | yes |
| light | Cyan line | `#0494A6` | glass (Full) | `#EEF6F5` | 3.3:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Cyan text | `#037685` | glass (Full) | `#EEF6F5` | 4.86:1 | 4.5:1 | active tab label, links | yes |
| light | Cyan line | `#0494A6` | glass dense | `#FAFCFC` | 3.52:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Cyan text | `#037685` | glass dense | `#FAFCFC` | 5.18:1 | 4.5:1 | active tab label, links | yes |
| both | on-accent on Amethyst | `#111114` | Amethyst fill | `#C27CF5` | 6.73:1 | 4.5:1 | button/chip label | yes |
| dark | Amethyst line | `#C27CF5` | bg | `#111114` | 6.73:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Amethyst text | `#C680F9` | bg | `#111114` | 7.05:1 | 4.5:1 | active tab label, links | yes |
| dark | Amethyst line | `#C27CF5` | surface | `#1B1B20` | 6.12:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Amethyst text | `#C680F9` | surface | `#1B1B20` | 6.42:1 | 4.5:1 | active tab label, links | yes |
| dark | Amethyst line | `#C27CF5` | glass (Full) | `#393042` | 4.47:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Amethyst text | `#C680F9` | glass (Full) | `#393042` | 4.69:1 | 4.5:1 | active tab label, links | yes |
| dark | Amethyst line | `#C27CF5` | glass dense | `#1D1C22` | 6.04:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Amethyst text | `#C680F9` | glass dense | `#1D1C22` | 6.33:1 | 4.5:1 | active tab label, links | yes |
| light | Amethyst line | `#AC67DE` | bg | `#F4EFE6` | 3.19:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Amethyst text | `#904ABF` | bg | `#F4EFE6` | 4.68:1 | 4.5:1 | active tab label, links | yes |
| light | Amethyst line | `#AC67DE` | surface | `#FFFFFF` | 3.65:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Amethyst text | `#904ABF` | surface | `#FFFFFF` | 5.36:1 | 4.5:1 | active tab label, links | yes |
| light | Amethyst line | `#AC67DE` | glass (Full) | `#F7F1F6` | 3.28:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Amethyst text | `#904ABF` | glass (Full) | `#F7F1F6` | 4.82:1 | 4.5:1 | active tab label, links | yes |
| light | Amethyst line | `#AC67DE` | glass dense | `#FDFBFC` | 3.55:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Amethyst text | `#904ABF` | glass dense | `#FDFBFC` | 5.2:1 | 4.5:1 | active tab label, links | yes |
| both | on-accent on Ice | `#111114` | Ice fill | `#8EC2FF` | 10.15:1 | 4.5:1 | button/chip label | yes |
| dark | Ice line | `#8EC2FF` | bg | `#111114` | 10.15:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Ice text | `#8EC2FF` | bg | `#111114` | 10.15:1 | 4.5:1 | active tab label, links | yes |
| dark | Ice line | `#8EC2FF` | surface | `#1B1B20` | 9.24:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Ice text | `#8EC2FF` | surface | `#1B1B20` | 9.24:1 | 4.5:1 | active tab label, links | yes |
| dark | Ice line | `#8EC2FF` | glass (Full) | `#323943` | 6.28:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Ice text | `#8EC2FF` | glass (Full) | `#323943` | 6.28:1 | 4.5:1 | active tab label, links | yes |
| dark | Ice line | `#8EC2FF` | glass dense | `#1C1D23` | 9.05:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Ice text | `#8EC2FF` | glass dense | `#1C1D23` | 9.05:1 | 4.5:1 | active tab label, links | yes |
| light | Ice line | `#5889C3` | bg | `#F4EFE6` | 3.17:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Ice text | `#3C6DA4` | bg | `#F4EFE6` | 4.68:1 | 4.5:1 | active tab label, links | yes |
| light | Ice line | `#5889C3` | surface | `#FFFFFF` | 3.63:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Ice text | `#3C6DA4` | surface | `#FFFFFF` | 5.36:1 | 4.5:1 | active tab label, links | yes |
| light | Ice line | `#5889C3` | glass (Full) | `#F4F6F7` | 3.35:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Ice text | `#3C6DA4` | glass (Full) | `#F4F6F7` | 4.95:1 | 4.5:1 | active tab label, links | yes |
| light | Ice line | `#5889C3` | glass dense | `#FCFCFC` | 3.54:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Ice text | `#3C6DA4` | glass dense | `#FCFCFC` | 5.23:1 | 4.5:1 | active tab label, links | yes |
| both | on-accent on Sand | `#111114` | Sand fill | `#E2BE8A` | 10.74:1 | 4.5:1 | button/chip label | yes |
| dark | Sand line | `#E2BE8A` | bg | `#111114` | 10.74:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Sand text | `#E2BE8A` | bg | `#111114` | 10.74:1 | 4.5:1 | active tab label, links | yes |
| dark | Sand line | `#E2BE8A` | surface | `#1B1B20` | 9.77:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Sand text | `#E2BE8A` | surface | `#1B1B20` | 9.77:1 | 4.5:1 | active tab label, links | yes |
| dark | Sand line | `#E2BE8A` | glass (Full) | `#3D3834` | 6.6:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Sand text | `#E2BE8A` | glass (Full) | `#3D3834` | 6.6:1 | 4.5:1 | active tab label, links | yes |
| dark | Sand line | `#E2BE8A` | glass dense | `#1D1D21` | 9.57:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Sand text | `#E2BE8A` | glass dense | `#1D1D21` | 9.57:1 | 4.5:1 | active tab label, links | yes |
| light | Sand line | `#A3814F` | bg | `#F4EFE6` | 3.16:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Sand text | `#856533` | bg | `#F4EFE6` | 4.69:1 | 4.5:1 | active tab label, links | yes |
| light | Sand line | `#A3814F` | surface | `#FFFFFF` | 3.61:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Sand text | `#856533` | surface | `#FFFFFF` | 5.38:1 | 4.5:1 | active tab label, links | yes |
| light | Sand line | `#A3814F` | glass (Full) | `#F9F5EF` | 3.33:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Sand text | `#856533` | glass (Full) | `#F9F5EF` | 4.95:1 | 4.5:1 | active tab label, links | yes |
| light | Sand line | `#A3814F` | glass dense | `#FDFCFA` | 3.52:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Sand text | `#856533` | glass dense | `#FDFCFA` | 5.24:1 | 4.5:1 | active tab label, links | yes |
| both | on-accent on Steel | `#111114` | Steel fill | `#9AA8BC` | 7.81:1 | 4.5:1 | button/chip label | yes |
| dark | Steel line | `#9AA8BC` | bg | `#111114` | 7.81:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Steel text | `#9AA8BC` | bg | `#111114` | 7.81:1 | 4.5:1 | active tab label, links | yes |
| dark | Steel line | `#9AA8BC` | surface | `#1B1B20` | 7.11:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Steel text | `#9AA8BC` | surface | `#1B1B20` | 7.11:1 | 4.5:1 | active tab label, links | yes |
| dark | Steel line | `#9AA8BC` | glass (Full) | `#33353B` | 5.08:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Steel text | `#9AA8BC` | glass (Full) | `#33353B` | 5.08:1 | 4.5:1 | active tab label, links | yes |
| dark | Steel line | `#9AA8BC` | glass dense | `#1C1C21` | 7.03:1 | 3.0:1 | bars, edges, sparklines | yes |
| dark | Steel text | `#9AA8BC` | glass dense | `#1C1C21` | 7.03:1 | 4.5:1 | active tab label, links | yes |
| light | Steel line | `#7A879A` | bg | `#F4EFE6` | 3.18:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Steel text | `#5F6C7E` | bg | `#F4EFE6` | 4.66:1 | 4.5:1 | active tab label, links | yes |
| light | Steel line | `#7A879A` | surface | `#FFFFFF` | 3.65:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Steel text | `#5F6C7E` | surface | `#FFFFFF` | 5.34:1 | 4.5:1 | active tab label, links | yes |
| light | Steel line | `#7A879A` | glass (Full) | `#F5F4F2` | 3.32:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Steel text | `#5F6C7E` | glass (Full) | `#F5F4F2` | 4.86:1 | 4.5:1 | active tab label, links | yes |
| light | Steel line | `#7A879A` | glass dense | `#FCFCFB` | 3.55:1 | 3.0:1 | bars, edges, sparklines | yes |
| light | Steel text | `#5F6C7E` | glass dense | `#FCFCFB` | 5.2:1 | 4.5:1 | active tab label, links | yes |
| dark | success fill/icon | `#34D399` | bg | `#111114` | 9.8:1 | 3.0:1 | status icon, dot, bar | yes |
| dark | success text | `#34D399` | bg | `#111114` | 9.8:1 | 4.5:1 | status word | yes |
| dark | success fill/icon | `#34D399` | surface | `#1B1B20` | 8.92:1 | 3.0:1 | status icon, dot, bar | yes |
| dark | success text | `#34D399` | surface | `#1B1B20` | 8.92:1 | 4.5:1 | status word | yes |
| dark | success fill/icon | `#34D399` | glass (Full) | `#403626` | 6.16:1 | 3.0:1 | status icon, dot, bar | yes |
| dark | success text | `#34D399` | glass (Full) | `#403626` | 6.16:1 | 4.5:1 | status word | yes |
| dark | success fill/icon | `#34D399` | glass dense | `#1E1C1F` | 8.8:1 | 3.0:1 | status icon, dot, bar | yes |
| dark | success text | `#34D399` | glass dense | `#1E1C1F` | 8.8:1 | 4.5:1 | status word | yes |
| light | success | `#047857` | bg | `#F4EFE6` | 4.79:1 | 4.5:1 | status word + icon | yes |
| light | success | `#047857` | surface | `#FFFFFF` | 5.48:1 | 4.5:1 | status word + icon | yes |
| light | success | `#047857` | glass (Full) | `#FBF5E8` | 5.05:1 | 4.5:1 | status word + icon | yes |
| light | success | `#047857` | glass dense | `#FEFCF8` | 5.35:1 | 4.5:1 | status word + icon | yes |
| dark | warning fill/icon | `#FF7A45` | bg | `#111114` | 7.29:1 | 3.0:1 | status icon, dot, bar | yes |
| dark | warning text | `#FE8759` | bg | `#111114` | 7.92:1 | 4.5:1 | status word | yes |
| dark | warning fill/icon | `#FF7A45` | surface | `#1B1B20` | 6.63:1 | 3.0:1 | status icon, dot, bar | yes |
| dark | warning text | `#FE8759` | surface | `#1B1B20` | 7.21:1 | 4.5:1 | status word | yes |
| dark | warning fill/icon | `#FF7A45` | glass (Full) | `#403626` | 4.58:1 | 3.0:1 | status icon, dot, bar | yes |
| dark | warning text | `#FE8759` | glass (Full) | `#403626` | 4.98:1 | 4.5:1 | status word | yes |
| dark | warning fill/icon | `#FF7A45` | glass dense | `#1E1C1F` | 6.54:1 | 3.0:1 | status icon, dot, bar | yes |
| dark | warning text | `#FE8759` | glass dense | `#1E1C1F` | 7.11:1 | 4.5:1 | status word | yes |
| light | warning | `#C2410C` | bg | `#F4EFE6` | 4.52:1 | 4.5:1 | status word + icon | yes |
| light | warning | `#C2410C` | surface | `#FFFFFF` | 5.18:1 | 4.5:1 | status word + icon | yes |
| light | warning | `#C2410C` | glass (Full) | `#FBF5E8` | 4.77:1 | 4.5:1 | status word + icon | yes |
| light | warning | `#C2410C` | glass dense | `#FEFCF8` | 5.05:1 | 4.5:1 | status word + icon | yes |
| dark | danger fill/icon | `#F87171` | bg | `#111114` | 6.81:1 | 3.0:1 | status icon, dot, bar | yes |
| dark | danger text | `#FF8482` | bg | `#111114` | 7.97:1 | 4.5:1 | status word | yes |
| dark | danger fill/icon | `#F87171` | surface | `#1B1B20` | 6.2:1 | 3.0:1 | status icon, dot, bar | yes |
| dark | danger text | `#FF8482` | surface | `#1B1B20` | 7.25:1 | 4.5:1 | status word | yes |
| dark | danger fill/icon | `#F87171` | glass (Full) | `#403626` | 4.28:1 | 3.0:1 | status icon, dot, bar | yes |
| dark | danger text | `#FF8482` | glass (Full) | `#403626` | 5.0:1 | 4.5:1 | status word | yes |
| dark | danger fill/icon | `#F87171` | glass dense | `#1E1C1F` | 6.12:1 | 3.0:1 | status icon, dot, bar | yes |
| dark | danger text | `#FF8482` | glass dense | `#1E1C1F` | 7.15:1 | 4.5:1 | status word | yes |
| light | danger | `#B91C1C` | bg | `#F4EFE6` | 5.65:1 | 4.5:1 | status word + icon | yes |
| light | danger | `#B91C1C` | surface | `#FFFFFF` | 6.47:1 | 4.5:1 | status word + icon | yes |
| light | danger | `#B91C1C` | glass (Full) | `#FBF5E8` | 5.95:1 | 4.5:1 | status word + icon | yes |
| light | danger | `#B91C1C` | glass dense | `#FEFCF8` | 6.31:1 | 4.5:1 | status word + icon | yes |
| dark | text | `#F4EFE6` | bg | `#111114` | 16.46:1 | 4.5:1 | body | yes |
| dark | muted | `#AFAAA0` | bg | `#111114` | 8.15:1 | 4.5:1 | secondary text | yes |
| dark | text | `#F4EFE6` | surface | `#1B1B20` | 14.98:1 | 4.5:1 | body | yes |
| dark | muted | `#AFAAA0` | surface | `#1B1B20` | 7.42:1 | 4.5:1 | secondary text | yes |
| dark | text | `#F4EFE6` | glass (Full) | `#403626` | 10.34:1 | 4.5:1 | body | yes |
| dark | muted | `#AFAAA0` | glass (Full) | `#403626` | 5.12:1 | 4.5:1 | secondary text | yes |
| dark | text | `#F4EFE6` | glass dense | `#1E1C1F` | 14.77:1 | 4.5:1 | body | yes |
| dark | muted | `#AFAAA0` | glass dense | `#1E1C1F` | 7.32:1 | 4.5:1 | secondary text | yes |
| light | text | `#111114` | bg | `#F4EFE6` | 16.46:1 | 4.5:1 | body | yes |
| light | muted | `#6B665C` | bg | `#F4EFE6` | 4.98:1 | 4.5:1 | secondary text | yes |
| light | mint text | `#047A55` | bg | `#F4EFE6` | 4.68:1 | 4.5:1 | secondary accent text | yes |
| light | text | `#111114` | surface | `#FFFFFF` | 18.85:1 | 4.5:1 | body | yes |
| light | muted | `#6B665C` | surface | `#FFFFFF` | 5.71:1 | 4.5:1 | secondary text | yes |
| light | mint text | `#047A55` | surface | `#FFFFFF` | 5.36:1 | 4.5:1 | secondary accent text | yes |
| light | text | `#111114` | glass (Full) | `#FBF5E8` | 17.35:1 | 4.5:1 | body | yes |
| light | muted | `#6B665C` | glass (Full) | `#FBF5E8` | 5.25:1 | 4.5:1 | secondary text | yes |
| light | mint text | `#047A55` | glass (Full) | `#FBF5E8` | 4.93:1 | 4.5:1 | secondary accent text | yes |
| light | text | `#111114` | glass dense | `#FEFCF8` | 18.39:1 | 4.5:1 | body | yes |
| light | muted | `#6B665C` | glass dense | `#FEFCF8` | 5.57:1 | 4.5:1 | secondary text | yes |
| light | mint text | `#047A55` | glass dense | `#FEFCF8` | 5.23:1 | 4.5:1 | secondary accent text | yes |
| dark | muted over sun glow | `#AFAAA0` | glass (Full) | `#403A2A` | 4.89:1 | 4.5:1 | secondary text | yes |
| light | muted over sun glow | `#6B665C` | glass (Full) | `#FBF6EA` | 5.29:1 | 4.5:1 | secondary text | yes |
| dark | muted over lime glow | `#AFAAA0` | glass (Full) | `#373E2A` | 4.82:1 | 4.5:1 | secondary text | yes |
| light | muted over lime glow | `#6B665C` | glass (Full) | `#F7F9EA` | 5.35:1 | 4.5:1 | secondary text | yes |
| dark | muted over rose glow | `#AFAAA0` | glass (Full) | `#402731` | 5.85:1 | 4.5:1 | secondary text | yes |
| light | muted over rose glow | `#6B665C` | glass (Full) | `#FBEDED` | 5.01:1 | 4.5:1 | secondary text | yes |
| dark | muted over cobalt glow | `#AFAAA0` | glass (Full) | `#293143` | 5.62:1 | 4.5:1 | secondary text | yes |
| light | muted over cobalt glow | `#6B665C` | glass (Full) | `#F0F2F7` | 5.09:1 | 4.5:1 | secondary text | yes |
| dark | text (Off) | `#F4EFE6` | surface | `#1B1B20` | 14.98:1 | 4.5:1 | body | yes |
| light | text (Off) | `#111114` | surface | `#FFFFFF` | 18.85:1 | 4.5:1 | body | yes |
| light | Saffron text (brief value) | `#8A5E00` | bg | `#F4EFE6` | 4.98:1 | 4.5:1 | brief reference | yes |
| light | Saffron line (brief value) | `#B87D00` | bg | `#F4EFE6` | 3.07:1 | 3.0:1 | brief reference | yes |
