# otvl_qstf

Golang module to develop streaming functions leveraging the QUIC Transport protocol
and interconnect them through an Application Proxy.

## Hosts, Connections, Streams and Functions

The Application Proxy (AP) is a QUIC server that is able to interconnect QUIC clients:
a QUIC client can connect to the AP and ask it to open a stream with another QUIC client,
in which case two streams will actually be created, one between each of the peers and the AP,
leaving to the AP the responsibility to relay the data chunks it reads from one to the other.

While a QUIC client can connect explicitly to an AP,
direct connections to a QUIC server are also authorized by the library,
anyway they will also implicitly enable the client to use the server as an AP.
This allows for simple scenarios such as direct callback from the server to the client,
but also for more complex collaborations from/to a remote peer through the AP.

The applications only deal with hosts, streams and functions.
The QUIC connections required to open QUIC streams are managed by the library,
a single connection is used for all streams between two given peers.
A connection accepted by a QUIC server (AP or requested server) enables the client to open streams,
and the server to open streams back on the client when requested to do so.

Streams are read-only, from the client to the server,
whose logical roles may be reversed when wanted thanks to the use of an AP.
A client can request a peer to open back a stream on itself
by executing a remote function for this purpose.
Streams are each created with an UUID, which may be pre-allocated or allocated on the fly.
Pre-allocated streams UUIDs are useful to get information back from a remote host,
for instance streaming the standard output of a remote executing command.

Stream opening requests provide a host-id, a stream-id and on option a function name.
The remote peer will accept a new QUIC stream and in the case the host is viewed through an AP,
it will wait for a connection from the target host and open a stream with the same UUID on it.

When a function name is provided, a corresponding function must be registered on the host.
It will be executed on stream acceptance,
with access to the underlying stream that typically transports the function input data.

## Implementation notes

### Client host-id

When connecting to a QUIC server for using it as an AP, a QUIC client has to explicitly call a function
`$Sys/IdentifyClient` that is intended to provide the server its host-id.
It is otherwise considered as anonymous and will not be authorized to request opening streams with remote peers.
The function `$Sys/IdentifyClient` also tells the AP if the client is accepting having streams opened on itself.

### Streams

Opened streams present a WriteCloser interface, accepted streams a Reader interface.
Both also present an interface to abort operations on the stream with an error code,
which unblock them immediately.
