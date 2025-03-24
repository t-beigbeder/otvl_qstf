# otvl_qstf

Golang module to develop streaming functions leveraging the QUIC Transport protocol
and interconnect them through an Application Proxy.

## Host/Connection/Stream/Function

The Application Proxy (AP) is a QUIC server that is able to interconnect QUIC clients:
a QUIC client can connect to the AP and ask it to open a stream with another QUIC client,
in which case two streams will in effect be created, one between each of the peers and the AP,
leaving to the AP the responsibility to relay read data chunks from one to the other.

While a QUIC client can connect explicitly to an AP,
direct connections to a QUIC server will actually enable clients
to leverage the server as an AP too,
enabling simple scenarios such as direct callback from the server to the client,
but more complex collaborations from/to a remote peer as well.

The applications only deal with hosts, streams and functions.
The required QUIC connections are managed by the library,
a single connection is used for all streams between two peers.
Connection accepted by the server enable the client to open streams on the server,
and the AP to open streams back on the client.

Streams are read-only, from the client to the server,
whose roles may be chosen as wanted thanks to the use of an AP.
A client can request a peer to open back a stream on itself
by executing a remote function for this purpose.
Streams are each created with an UUID, which may be pre-allocated or allocated on the fly.
Pre-allocated streams UUIDs are useful to get information back from a remote host,
for instance a remote executing command standard output.

Stream opening requests provide a host-id, a stream-id and on option a function name.
The remote peer will accept a new QUIC stream and in the case the host is viewed through a proxy,
wait for a connection from the host and open a stream with the same UUID on it.

When a function name is provided, a corresponding function must be registered on the host.
It will be executed on stream acceptance,
with access to the underlying stream that typically transports the function input data.
