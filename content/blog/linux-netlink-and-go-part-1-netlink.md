+++
date = "2017-02-21T00:00:00+00:00"
title = "Linux, Netlink, and Go - Part 1: netlink"
subtitle = "An introduction to Linux's netlink subsystem, and a tutorial on how to make use of it with Go."
+++

I'm a big fan of [Prometheus](https://prometheus.io/). I use it quite a lot at
both home and work, and greatly enjoy having insight into what my systems are
doing at any given moment. One of the most widely used Prometheus exporters is
the [node_exporter](https://github.com/prometheus/node_exporter): a daemon that
can extract a wide variety of metrics from UNIX-like machines.

As I was browsing the repository, I noticed an open issue requesting the addition
of WiFi metrics to node_exporter. The idea intrigued me, and I realized that I
would certainly make use of such a feature on my Linux laptop. I began exploring
options for retrieving WiFi device information on Linux.

After a couple of weeks of experimentation (including the legacy `ioctl()`
wireless extensions API), I authored three Go packages which work together to
interact with WiFi devices on Linux:

- [`netlink`](https://github.com/mdlayher/netlink): provides low-level access
  to Linux netlink sockets.
- [`genetlink`](https://github.com/mdlayher/genetlink): implements generic
  netlink interactions and data types.
- [`wifi`](https://github.com/mdlayher/wifi): provides access to IEEE 802.11
  WiFi device actions and statistics.

This series of posts will describe some of the lessons I learned while
implementing these packages in Go, and hopefully provide a nice reference for
others who wish to experiment with netlink and/or WiFi devices in their language
of choice.

The pseudo-code in this series will use Go's [`golang.org/x/sys/unix`](https://godoc.org/golang.org/x/sys/unix)
package and types from my `netlink`, `genetlink`, and `wifi` packages. The
series is broken up as follows:

- [Part 1: netlink](.) (this post): an introduction to netlink.
- [Part 2: generic netlink](../linux-netlink-and-go-part-2-generic-netlink): an
  introduction to generic netlink, a netlink family meant to simplify creation
  of new families.
- [Part 3: packages netlink, genetlink, and wifi](../linux-netlink-and-go-part-3-packages-netlink-genetlink-and-wifi):
  using Go to drive interactions with netlink, generic netlink, and nl80211.

## What is netlink?

[Netlink](https://en.wikipedia.org/wiki/Netlink) is a Linux kernel inter-process
communication mechanism, enabling communication between a userspace process and
the kernel, or multiple userspace processes. Netlink sockets are the primitive
which enables this communication.

This post will provide a primer on netlink sockets, messages, multicast groups,
and attributes. In addition, this post will focus on communication between
userspace and the kernel, rather than communication between two userspace
processes.

## Creating netlink sockets

Netlink makes use of the standard [BSD sockets API](https://en.wikipedia.org/wiki/Berkeley_sockets),
which is typically used for network programming in C. If you'd like to learn more
about BSD sockets, I recommend the excellent [Beej's Guide to Network Programming](http://beej.us/guide/bgnet/)
for a primer on the topic.

It is important to note that **netlink communications never traverse beyond the local host**.
With this in mind, let's begin diving into how netlink sockets work!

To communicate with netlink, a netlink socket must be opened. This is done
using the `socket()` system call:

```go
fd, err := unix.Socket(
    // Always used when opening netlink sockets.
    unix.AF_NETLINK,
    // Seemingly used interchangeably with SOCK_DGRAM,
    // but it appears not to matter which is used.
    unix.SOCK_RAW,
    // The netlink family that the socket will communicate
    // with, such as NETLINK_ROUTE or NETLINK_GENERIC.
    family,
)
```

The family parameter specifies a particular netlink family: essentially, a
kernel subsystem which can be communicated with using netlink sockets. These
families may offer functionality such as:

- `NETLINK_ROUTE`: manipulation of Linux's network interfaces, routes, IP
  addresses, etc.
- `NETLINK_GENERIC`: a building block for simplified addition of new netlink
  families, like nl80211, Open vSwitch, etc.

Once the socket is created, `bind()` must be called to prepare it to send and
receive messages.

```go
err := unix.Bind(fd, &unix.SockaddrNetlink{
    // Always used when binding netlink sockets.
    Family: unix.AF_NETLINK,
    // A bitmask of multicast groups to join on bind.
    // Typically set to zero.
    Groups: 0,
    // If you'd like, you can assign a PID for this socket
    // here, but in my experience, it's easier to leave
    // this set to zero and let netlink assign and manage
    // PIDs on its own.
    Pid: 0,
})
```

At this point, the netlink socket is now ready to send and receive messages to
and from the kernel.

## Netlink message format

Netlink messages follow a very particular format. All messages must be aligned
to a 4 byte boundary. As an example, a 16 byte message **must be sent as is**,
but a 17 byte message **must be padded** to 20 bytes.

It is very important to note that, unlike typical network communications,
netlink uses the **host byte order**, or [endianness](https://en.wikipedia.org/wiki/Endianness),
for encoding and decoding integers, instead of the common network byte order
(big endian). As a result, code which must convert between byte and integer
representations of data must keep this in mind.

Netlink message headers make use of the following format: (diagram from
[RFC 3549](https://tools.ietf.org/html/rfc3549#section-2.3.2)):

```plaintext
0                   1                   2                   3
0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                          Length                             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|            Type              |           Flags              |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                      Sequence Number                        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                      Process ID (PID)                       |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

These fields contain the following information:

- **Length** (32 bits): the length of the entire message, including both headers
  and payload.
- **Type** (16 bits): what kind of information the message contains, such as an
  error, end of multi-part message, etc.
- **Flags** (16 bits): bit flags which indicate that a message is a request, a
  multi-part message, an acknowledgement of a request, etc.
- **Sequence Number** (32 bits): a number used to correlate requests and responses;
  incremented on each request.
- **Process ID (PID)** (32 bits): sometimes referred to as port ID; a number used
  to uniquely identify a particular netlink socket; may or may not be the
  process's ID.

Finally, a payload may immediately follow a netlink header. Again, note that the
payload must be padded to a 4 byte boundary.

An example netlink message which sends a request to the kernel may resemble the
following in Go:

```go
msg := netlink.Message{
    Header: netlink.Header{
        // Length of header, plus payload.
        Length: 16 + 4,
        // Set to zero on requests.
        Type: 0,
        // Indicate that message is a request to the kernel, and we expect
        // an acknowledgement in return.
        Flags: netlink.Request | netlink.Acknowledge,
        // Sequence number selected at random.
        Sequence: 1,
        // PID set to process's ID.
        PID: uint32(os.Getpid()),
    },
    // An arbitrary byte payload. May be in a variety of formats.
    Data: []byte{0x01, 0x02, 0x03, 0x04},
}
```

## Sending and receiving netlink messages

Now that we are familiar with some of the basics of netlink sockets, we can
send and receive data using a socket.

Once a message has been prepared, it can be sent to the kernel using `sendto()`:

```go
// Assume messageBytes produces a netlink request message (like the
// one shown above) with the specified payload.
b := messageBytes([]byte{0x01, 0x02, 0x03, 0x04})
err := unix.Sendto(b, 0, &unix.SockaddrNetlink{
    // Always used when sending on netlink sockets.
    Family: unix.AF_NETLINK,
})
```

Read-only requests to netlink typically do not require any special privileges.
Operations which modify the state of a subsystem using netlink, or require
locking its internal state, typically require elevated privileges. This may
mean running the program as root or using [`CAP_NET_ADMIN`](http://man7.org/linux/man-pages/man7/capabilities.7.html)
to:

- Send a write request to make changes to a subsystem using netlink.
- Send a read request with the `NLM_F_ATOMIC` flag, to receive an atomic
  snapshot of data from netlink.

Receiving messages from a netlink socket using `recvfrom()` can be slightly more
complicated, depending on a variety of factors. Netlink may reply with:

- Very small or very large messages.
- Multi-part messages, broken into multiple pieces.
- An explicit error number, when header type is "error".

In addition, the sequence number and PID of each message should be validated
as well. When working with raw system calls, it's up to the socket's user to
handle these cases.

## Large messages

To deal with large messages, I've employed a technique of allocating a single
page of memory, peeking at the buffer (without draining it), and then doubling
the size of the buffer if it's too small to read the entire message. Thanks,
[Dominik Honnef](https://github.com/dominikh) for your insight on this problem.

```go
b := make([]byte, os.Getpagesize())
for {
    // Peek at the buffer to see how many bytes are available.
    n, _, err := unix.Recvfrom(fd, b, unix.MSG_PEEK)
    if err != nil {
        return nil, err
    }

    // Break when we can read all messages.
    if n < len(b) {
        break
    }

    // Double in size if not enough bytes.
    b = make([]byte, len(b)*2)
}

// Read out all available messages.
n, _, err := unix.Recvfrom(fd, b, 0)
```

In theory, a netlink message may be of a size up to ~4GiB (maximum 32-bit
unsigned integer), but in practice, messages are much smaller.

## Multi-part messages

For certain types of messages, netlink may reply with a "multi-part message".
In this case, each message before the final one will have the "multi" flag set.
The final message will have a type of "done".

When returning multi-part messages, the first `recvfrom()` will return all
messages with the "multi" flag set. Next, `recvfrom()` **must be called again**
to retrieve the final message with header type "done". This is very important or
else netlink will simply hang on subsequent requests, waiting for the caller to
drain the final header type "done" message.

The code for this isn't as trivial as other examples, but you can take a look
at [my implementation](https://github.com/mdlayher/netlink/blob/1c1ce40bf284f4af7cecfe578a9d3276536a2b2d/conn.go#L274)
if you'd like a reference.

## Netlink error numbers

If netlink cannot satisfy a request for whatever reason, it will return an
explicit error number in the payload of a message containing header type
"error". These error numbers are the same as Linux's classic error numbers, such
as `ENOENT` for "no such file or directory", or `EPERM` for "permission denied".

If a message's header type indicates an error, the error number will be encoded
as a signed 32 bit integer (note: also uses system endianness) in the first 4
bytes of the message's payload.

```go
const name = "foo0"
_, err := rtnetlink.InterfaceByName(name)
if err != nil && os.IsNotExist(err) {
    // Error is result of a netlink error number, and can be
    // checked in the usual Go fashion.
    log.Printf("no such device: %q", name)
    return
}
```

## Sequence number and PID validation

To ensure a netlink reply from the kernel is in response to one of our requests,
we must also validate the sequence number and PID on each received message. In
the majority of cases, these should match exactly what was sent to the kernel
with a request. Subsequent requests should increment the sequence number before
sending another message to netlink.

PID validation may vary slightly, depending on several conditions.

- If a message is received in userspace on behalf a multicast group, it will
  have a PID of 0, meaning the message originated in the kernel.
- If a request is sent to the kernel with a PID of 0, netlink will assign a PID
  for a given socket on the first response. This PID should be used (and
  validated) in subsequent communications.

Assuming you didn't specify a PID in `bind()`, when opening multiple netlink
sockets in a single application, the first one will be assigned a PID of the
process's ID. Subsequent ones will have a random number chosen by netlink. In my
experience, it is much easier to just **let netlink assign all PIDs** itself,
and make sure you keep track of which numbers it assigns for each socket.

## Multicast groups

In addition to the classic request/response socket paradigm, netlink sockets
also provide multicast groups to enable subscribing to certain events as they
occur.

A multicast group can be joined using two different methods:

- Specifying a groups bitmask during `bind()`. This is considered the "legacy"
  method.
- Joining and leaving groups using `setsockopt()`. This is the preferred, modern
  method.

Joining and leaving groups using `setsockopt()` is a matter of swapping a single
constant. In Go, this is done using `uint32` "group" values.

```go
// Can also specify unix.NETLINK_DROP_MEMBERSHIP to leave
// a group.
const joinLeave = unix.NETLINK_ADD_MEMBERSHIP
// Multicast group ID. Typically assigned using predefined
// constants for various netlink families.
const group = 1

err := unix.SetSockoptInt(
    fd,
    unix.SOL_NETLINK,
    joinLeave,
    group,
)
```

Once a group is joined, you can listen for messages using `recvfrom()` as usual.
Leaving the group will cause no further messages to be delivered for a given
multicast group.

## Netlink attributes

To wrap up our primer on netlink sockets, we will discuss a very common data
format for netlink message payloads: attributes.

Netlink attributes are unusual in that they are in **LTV (length, type, value)**
format, instead of the typical TLV (type, length, value). As with every other
integer in netlink sockets, the type and length values are also encoded with
**host endianness**. Finally, netlink attributes **must also be padded** to a 4
byte boundary, just like netlink messages.

Each field contains the following information:

- **Length** (16 bits): the length of the entire attribute, including length, type
  and value fields. May not be set to a 4 byte boundary. For example, if length
  is 17 bytes, the attribute will be padded to 20 bytes, but the 3 bytes of
  padding should not be interpreted as meaningful.
- **Type** (16 bits): the type of an attribute, typically defined as a constant
  in some netlink family or header.
- **Value** (variable bytes): the raw payload of an attribute. May contain nested
  attributes, which are stored in the same format. Those nested attributes may
  contain even more nested attributes!

There are two special flags which may be present in netlink attributes, though
I have yet to encounter them in my work.

- `NLA_F_NESTED`: specifies a nested attribute; used as a hint for parsing.
  Doesn't always appear to be used, even if nested attributes are present.
- `NLA_F_NET_BYTEORDER`: attribute data is stored in network byte order (big
  endian) instead of host endianness.

Consult the documentation of a given netlink family to determine if either of
these flags should be checked.

## Summary

Now that we are familiar with using netlink sockets and messages, the next post
in the series will build upon this knowledge to dive into generic netlink.

Hope you enjoyed this post! If you have questions or comments, feel free to
reach out via [Twitter](https://twitter.com/mdlayher), or [Gophers Slack](https://invite.slack.golangbridge.org/)
(username: mdlayher).

## References

The following links were used frequently as a reference as I built out package
netlink, and authored this post:

- [Wikipedia: Netlink](https://en.wikipedia.org/wiki/Netlink)
- [Communicating between the kernel and user-space in Linux using Netlink sockets](https://pdfs.semanticscholar.org/6efd/e161a2582ba5846e4b8fea5a53bc305a64f3.pdf)
- [Understanding And Programming With Netlink Sockets](https://people.redhat.com/nhorman/papers/netlink.pdf)
- [Netlink (C) Library (libnl)](https://www.infradead.org/~tgr/libnl/doc/core.html)
