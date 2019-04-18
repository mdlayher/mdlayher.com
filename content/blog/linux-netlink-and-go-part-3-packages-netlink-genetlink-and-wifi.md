+++
date = "2017-03-13T00:00:00+00:00"
title = "Linux, Netlink, and Go - Part 3: packages netlink, genetlink, and wifi"
subtitle = "Using netlink, generic netlink, and nl80211 to manipulate WiFi network interfaces on Linux, using Go."
+++

In [Part 1](../linux-netlink-and-go-part-1-netlink) and [Part 2: generic netlink](../linux-netlink-and-go-part-2-generic-netlink)
of this series, I described some of the fundamental concepts of netlink and
generic netlink. It is assumed that readers are already familiar with netlink
and generic netlink from the previous posts in this series.

In this post, I will dive into high level concepts and usage of my Go packages:

- [`netlink`](https://github.com/mdlayher/netlink): provides low-level access
  to Linux netlink sockets.
- [`genetlink`](https://github.com/mdlayher/genetlink): implements generic
  netlink interactions and data types.
- [`wifi`](https://github.com/mdlayher/wifi): provides access to IEEE 802.11
  WiFi device actions and statistics.

The series is split into parts as follows:

- [Part 1: netlink](../linux-netlink-and-go-part-1-netlink): an introduction
  to netlink.
- [Part 2: generic netlink](../linux-netlink-and-go-part-2-generic-netlink): an
  introduction to generic netlink, a netlink family meant to simplify creation
  of new families.
- [Part 3: packages netlink, genetlink, and wifi](.): using Go to drive
  interactions with netlink, generic netlink, and nl80211.

## Package netlink

When I first decided to look into retrieving WiFi device information on Linux,
I was pointed to netlink by a colleague as a possible solution. I was familiar
with several existing netlink packages for Go, but I was unable to find one that
I felt suited my needs.

Several of the existing, popular packages made decisions I wasn't happy with:

- lack of documentation
- lack of tests
- bloated API
- a "C-like" Go API

In addition, many of these packages conflated the concepts of "netlink",
"route netlink", and the [iproute2](https://en.wikipedia.org/wiki/Iproute2)
family of utilities. For these reasons, I decided to start building my own
package.

It is important to note that package [`netlink`](https://github.com/mdlayher/netlink)
is meant to be used as a **building block for other netlink family packages**.
Typically, you will use a higher level package like
[`genetlink`](https://github.com/mdlayher/genetlink) instead. To enable maximum
code re-use, create dedicated packages for any high level netlink families,
instead of building them on top of package `netlink` in your application.

This section will discuss some common use-cases of package netlink, but you may
wish to reference the [source code](https://github.com/mdlayher/netlink) and
[documentation](https://godoc.org/github.com/mdlayher/netlink) for further
information.

## netlink.Conn: a connection to netlink

The `netlink.Conn` type is used to create connections to netlink. A netlink
family is specified in the call to `netlink.Dial` along with additional
configuration, if needed.

It is important to note that when a `netlink.Conn` is no longer needed, the
`Close` method must be called to close the socket and avoid leaking
file descriptors.

Once a connection is established, it can be used to send a request, receive a
response, and validate the response against the request. Many of the request
header fields can be omitted to allow package `netlink` to calculate and assign
those values automatically.

Error checking omitted for brevity. Please check all errors in your code.

```go
// Dial generic netlink.
const genetlink = 16
conn, _ := netlink.Dial(genetlink, nil)
defer conn.Close()

m := netlink.Message{
    Header: netlink.Header{
        // Ask netlink to echo back an acknowledgement to our request.
        Flags: netlink.Request | netlink.Acknowledge
        // Other fields assigned automatically by package netlink.
    },
}

// Perform a send, receive, and validate cycle.
req, _ := conn.Send(m)
replies, _:= conn.Receive()
err := netlink.Validate(req, replies)
```

For convenience, the `Execute` method can be used as a shortcut for `Send`,
`Receive`, and `netlink.Validate`.

```go
msgs, _ := conn.Execute(m)
```

To listen to multicast groups, use the `JoinGroup` and `LeaveGroup` methods.
Messages can be received using `Receive` as usual.

```go
// Listen to rtnetlink for modification of network interfaces
const rtnetlink = 0
const rtmGroupLink = 0x1

conn, _ := netlink.Dial(rtnetlink, nil)
defer conn.Close()

// Join multicast group: Receive will block until messages arrive.
_ = conn.JoinGroup(rtmGroupLink)
msgs, _ := conn.Receive()
_ = conn.LeaveGroup(rtmGroupLink)
```

Finally, when dealing with a high throughput netlink application, one may wish
to make use of BPF filters to explicitly accept or reject packets based on
their contents.

BPF filters can be attached by calling `SetBPF` with a filter assembled using
[`golang.org/x/net/bpf`](https://godoc.org/golang.org/x/net/bpf).

## Package genetlink

Package `genetlink` is the reference example of a netlink family package built
using package `netlink`. It exposes a very similar API, but handles some common
generic netlink operations on behalf of its user.

This package enables sending and receiving netlink messages but also offers
convenience methods to retrieve generic netlink families from `nlctrl`, the
generic netlink controller.

If you'd like to dive in, you may reference the [source code](https://github.com/mdlayher/genetlink)
and [documentation](https://godoc.org/github.com/mdlayher/genetlink) for further
information.

## genetlink.Conn: a connection to generic netlink

A `genetlink.Conn` is essentially a specialized wrapper around the `netlink.Conn`
type for interacting with generic netlink. It transparently adds and removes
`netlink.Message` types where needed, and enables the caller to only use
`genetlink.Message` in most cases.

As with a `netlink.Conn`, when a `genetlink.Conn` is no longer needed, the
`Close` method must be called to close the socket and avoid leaking file
descriptors.

The methods `Send`, `Receive`, and `Execute` in package `genetlink` are used to
work with generic netlink messages. In fact, when using `Execute`, there is no
need to deal with any `netlink.Message` types at all.

```go
conn, _ := genetlink.Dial(nil)
defer conn.Close()

const (
    ctrlVersion = 1
    ctrlCommandGetFamily = 3
)

// Ask nlctrl to list all known families.
req := genetlink.Message{
    Header: genetlink.Header{
       Command: ctrlCommandGetFamily,
       Version: ctrlVersion,
    },
}

flags := netlink.Request | netlink.Dump
msgs, _ := conn.Execute(req, genetlink.Controller, flags)
```

Finally, the methods `JoinGroup`, `LeaveGroup`, and `SetBPF` are all available
with `genetlink.Conn`. They perform the same actions as they do with a
`netlink.Conn`.

## genetlink.Family: generic netlink families

Because querying family information from the generic netlink controller is so
common, `genetlink.Conn` provides specialized methods and types for doing so.

The `genetlink.Family` type provides information about a given generic netlink
family, including its ID, version, name, and multicast groups.

The `ListFamilies` and `GetFamily` methods can be used to retrieve generic
netlink families from `nlctrl`, the generic netlink controller.

```go
conn, _ := genetlink.Dial(nil)
defer conn.Close()

// Ask if nl80211 is available on this system. If it is not,
// an error compatible with netlink.IsNotExist is returned.
if _, err := conn.GetFamily("nl80211"); netlink.IsNotExist(err) {
    fmt.Println("nl80211 not available")
    return
}
```

As discussed in Part 2 of this series, communicating with a generic netlink
family involves specifying its family ID and version in a request:

```go
conn, _ := genetlink.Dial(nil)
defer conn.Close()

f, _ := conn.GetFamily("nl80211")

// Ask nl80211 for a list of all WiFi interfaces.
req := genetlink.Message{
    Header: genetlink.Header{
        Command: nl80211.CmdGetInterface,
        // Specify the version of nl80211 we are speaking.
        Version: uint8(f.Version)
    },
}

// Specify the ID of nl80211 in call to Execute.
flags := netlink.Request | netlink.Dump
msgs, _ := conn.Execute(req, f.ID, flags)
```

## Package wifi

Package `wifi` provides access to IEEE 802.11 WiFi device actions and statistics.
At this time, it only works with Linux, though I'd love to incorporate support
for more operating systems in the future.

If you'd like to dive in, you may reference the [source code](https://github.com/mdlayher/wifi)
and [documentation](https://godoc.org/github.com/mdlayher/wifi) for further
information.

On Linux, package `wifi` works using nl80211: a generic netlink family that
provides C kernel header definitions for all of its commands and netlink
attributes. To easily create a Go package from these constants, I made use of
the excellent [c-for-go tool](https://github.com/xlab/c-for-go) by Maxim
Kupriianov. Maxim was even kind enough to provide the initial generated code
for working with nl80211 from Go.

## wifi.Client: accessing WiFi devices from Go

Because of the foundation provided by packages netlink and genetlink, usage of
package `wifi` on Linux can be made very concise and clean. Keep in mind that
the `Close` method must be called when the `wifi.Client` is longer needed, in
order to clean up the underlying netlink socket.

This example retrieves a list of all WiFi-enabled network interfaces, and
fetches the SSID associated with each device:

```go
client, _ := wifi.New()
defer client.Close()

ifis, _ := client.Interfaces()
for _, ifi := range ifis {
    // For more information about what a "BSS" is, see:
    // https://en.wikipedia.org/wiki/Service_set_(802.11_network).
    bss, _ := client.BSS(ifi)
    fmt.Printf("%s: %q\n", ifi.Name, bss.SSID)
}
```

That's really all there is to it! Netlink, generic netlink, and nl80211 do the
heavy lifting of requesting and retrieving data from the kernel, decoding it,
and packaging it up nicely for the user.

## Summary

That wraps up my series on Linux, Netlink, and Go! I hope you've enjoyed the
series, and found it informative. If you'd like to get started working with
netlink, I'd encourage you to check out:

- [the importers list of package `netlink`](https://godoc.org/github.com/mdlayher/netlink?importers)
- [my GopherCon 2018 lightning talk](https://www.youtube.com/watch?v=tw-9fNygYE4)
- [the slides from my lightning talk](https://github.com/mdlayher/talks/blob/master/gophercon2018/linux-netlink-and-go.pdf)

If you'd like to begin work on a new netlink family package, I'd love to hear
from you. I happily welcome contributions to all of the packages discussed in
this series. Please file an issue if you'd like to contribute!

If you have questions or comments, feel free to reach out via [Twitter](https://twitter.com/mdlayher)
or [Gophers Slack](https://invite.slack.golangbridge.org/) (username: mdlayher).

Thank you very much for your time. It's been a pleasure authoring this series,
and I've received some great feedback from a wide variety of readers.
I'll keep writing if you keep reading! Until next time!
