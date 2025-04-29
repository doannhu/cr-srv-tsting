Application          SOP gRPC server
┌───────────── ┐      ┌─────────────────┐
│ create ctx   │      │                 │
│ tlsCfg       │      │                 │
│ retryCfg     │      │                 │
│ NewSOPClient │──TLS handshake────────►│
│             ◄──────connection ready───┘
│ ... later ...│
│ GetConsolidatedSOP() ───RPC──────────►│
│ (Retry helper wraps) │                │
│◄──response / error───┘                │
└─────────────┘

1. Security first – refuses to start without certificates.

2. Fail-fast connection – problems explode during startup, not on first business request.

3. Clean interface for business code – only GetConsolidatedSOP is exposed.

4. Isolated retry logic – one helper (utils.Retry) centralises back-off rules.