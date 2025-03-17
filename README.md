# otvl_qstf

Golang module to develop streaming functions leveraging the QUIC Transport protocol
and interconnect them through an Application Proxy.

## Foundation

Based on [go-streams](https://github.com/reugn/go-streams) for the pipelines
and on [quic-go](https://quic-go.net/docs/) for the implementation of the protocol.

## Connection/Stream

On a QUIC connection, open a standard stream and provide (streamId, hostId, funcName):

- using direct connection, after peer host accepts stream opening, peer starts monitoring streamUuid
and optionally executes a function registered by its name,
function is implicitly reading the stream (as a flow.Map connected to it)
- using application proxy, enables to ask the AP to request the same on an actual target host
- local use of opened streams except when initializing is standard
- peer's accepted streams are not handled directly as quic.xxxStream by the application
but as qstf.Stream that are unregistered properly on EOF

Functions are not monitored, it is their implementation's responsibility to release resources
on stream close. They are executed with a dedicated child context of the connection's one.

A client can request a peer to open back a stream by executing a remote function that does the same.

A client can open additional streams and communicate their streamIds
to the target function on its stream.
The protocol between the client and the target function is not constrained.

### Application Proxy protocol

A connecting host informs the AP about its identity (hostId) by calling a predefined function
named QstfRegisterHost. It opens and accepts connections with the AP.
The AP opens connection to the target peer requested with first stream opening.
There is one connection to target host per origin host.
Each connection is monitored by the AP and other end connection is closed accordingly.

Streams accepted are propagated by opening them on the other end.
Streaming data is opaque and sent to the other end as soon as it is received.
It may be encrypted end-to-end.
