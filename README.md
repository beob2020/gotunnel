# Prooduction-Ready Go Tunnel
 A tunneling tool in Go that exposes your local network to the internet. 
This solution includes enterprise-grade features like TLS encryption, authentication, rate limiting, observability, and graceful shutdown.

# Architecture Overview
The tool uses a client-server architecture:
- Server: Public-facing component that accepts client connections and proxies traffic.
- Client: Runs on your local network, establishes outbound connections to the server.
- Protocol: WebSocket with TLS for initial connection, then Yamux for multiplexing.
- Security: mutual TLS, JWT authentication, rate limiting, and IP whitelisting.
