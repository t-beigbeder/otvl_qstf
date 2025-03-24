# otvl_qstf

Golang module to develop streaming functions leveraging the QUIC Transport protocol
and interconnect them through an Application Proxy.

## Host/Connection/Stream

The Application Proxy (AP) is a QUIC server that is able to interconnect QUIC clients:
a QUIC client can connect to the AP and ask it to open a stream with another QUIC client,
two streams will in effect be created, one between each of the peers and the AP,
leaving to the AP the responsibility to relay read data chunks from one to the other.

QUIC connections are managed by the library.
The applications only deal with hosts, streams and functions.

While a QUIC client can connect explicitly to an Application Proxy (AP),
direct connections to a QUIC server will actually enable corresponding clients
to leverage it as an AP too,
enabling simple scenarios such as direct callback from the server to the client,
but more complex collaborations as well.

[...] 

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
