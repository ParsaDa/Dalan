# Dalan 0.6.0 — release notes (draft)

## Breaking

- **Xray tunnels stop working.** The Xray (REALITY) tunnel engine was removed; tunnels run only on BackPack and Rakhsh. On the first start Dalan deletes every Xray tunnel, the management tunnels that rode on them and their places in spare pools. Make a BackPack or Rakhsh tunnel to replace each one.
- **Nodes enrolled over an Xray tunnel must enrol again.** They stay in the node list with a "must be enrolled again" notice. Press **Enrol again** and run the new command on the node; its inbounds and history are kept. The node needs its engine binary first (`x-ui install-engines`).
- Setup links of version 1 (Xray) and enrolment links of version 1 are refused.
- The API drops `security`, `serverName`, `realityPublicKey` and `shortId` from tunnels, and `security`, `serverName` and `withTunnel` from `POST /panel/api/enrolments/add`, which takes `engine`, `transport` and `tunnelId` instead.

## New

- Adding a node with a command asks for the engine (BackPack by default, or Rakhsh) and for a new tunnel for the node or an existing entry tunnel.
