# otvl_qstf

Golang module to develop streaming functions leveraging the QUIC Transport protocol
and interconnect them through an Application Proxy.

## URLs

otvlstf+quic://host:port
otvlstf+unix:///path/to/socket.sock

## Connection/Stream abstraction

Could be in-process pipes, unix domain sockets or QUIC connections/streams.

On a pipe/socket, open as many secondary channels as there are streams.

On a QUIC connection, open a control stream:

- using application proxy, enables to request an actual target host for accepting a stream
- using direct connection, enables to request the peer host for accepting a stream
