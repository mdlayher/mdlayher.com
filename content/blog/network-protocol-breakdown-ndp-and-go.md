+++
date = "2018-05-30T08:00:00+00:00"
title = "Network Protocol Breakdown: NDP and Go"
subtitle = "An introduction to the IPv6 Neighbor Discovery protocol, and how to utilize it from Go."
+++

If you've ever studied the fundamentals of computer networks, you may be
familiar with the [Address Resolution Protocol](https://en.wikipedia.org/wiki/Address_Resolution_Protocol),
or "ARP". ARP is a crucial piece of IPv4 designed to allow computers on the same
network to discover each other's MAC addresses using their IPv4 addresses. IPv4
itself remains a crucial component of computer networks today, but how does IPv6
perform this task?

This is where the [Neighbor Discovery Protocol](https://en.wikipedia.org/wiki/Neighbor_Discovery_Protocol),
"NDP" (sometimes just "ND"), comes into play. You can think of NDP as a more
flexible, IPv6-only, modern version of ARP. It performs all of the same
functions that ARP does for IPv4 while also enabling some interesting new
functionality for IPv6.

In this blog, you will learn about some of the fundamentals of NDP, how NDP is
used with IPv6, and how to work with NDP using the Go programming language.

![image](/img/blog/network-protocol-breakdown-ndp-and-go/1.png)
*Using the `ndp` tool (<https://github.com/mdlayher/ndp>) to send Router Solicitation messages.*

## An introduction to NDP

NDP is specified in [IETF RFC 4861](https://tools.ietf.org/html/rfc4861). In
many ways, NDP is the spiritual successor to ARP for IPv6. It is used to help
computers find the MAC addresses of their neighbors on a local network, but it
also has much more to offer thanks to its added flexibility and extensibility.

To start, machines that make use of IPv6 always assign a [link-local IPv6 address](https://en.wikipedia.org/wiki/Link-local_address#IPv6)
to each of their enabled network interfaces. Addresses in this block reside in
`fe80::/10`, and can often be easily identified by their prefix alone. Because
of the existence of this address, NDP can be transported over [ICMPv6](https://en.wikipedia.org/wiki/Internet_Control_Message_Protocol_for_IPv6)
and IPv6. The ICMPv6 packet header contains:

- **Type:** a general class of messages (indicates NDP message types).
- **Code:** a sub-class of messages within a type (always 0 for NDP).
- **Checksum:** used to verify the integrity of an ICMPv6 message.
- **Data:** an arbitrary payload (contains NDP message structures).

![image](/img/blog/network-protocol-breakdown-ndp-and-go/2.png)
*An ICMPv6 header. NDP uses ICMPv6 for transport.*

NDP has several different message types which allow it to be used for
discovering IPv6 neighbors and routers on a local network. These include:

- **Neighbor Solicitation:** ask a neighbor for its MAC address using its
  IPv6 address.
- **Neighbor Advertisement:** inform a neighbor of an interface's MAC address.
- **Router Solicitation:** request that routers generate router advertisement
  messages.
- **Router Advertisement:** inform neighbors that a router is available to be
  used as a default IPv6 router.
- **Redirect:** sent by a router in response to an IPv6 packet to inform an
  interface of a better first hop router. This message won't be covered in
  detail in this post.

In addition, NDP supports options in a flexible TLV (type, length, value)
format, making the protocol extremely extensible. Some commonly used options
include:

- **Source/Target link-layer addresses:** the source or target (depending on
  solicitation/advertisement messages) link-layer address of an IPv6-speaking
  interface.
- **Prefix information:** sent by routers to inform neighbors if an IPv6 prefix
  is "on-link", and if it can be used for [Stateless Address Autoconfiguration (SLAAC)](https://en.wikipedia.org/wiki/IPv6#Stateless_address_autoconfiguration_%28SLAAC%29).
- **MTU:** sent by routers to inform neighbors of the expected MTU on a network.
- **Recursive DNS servers (RDNSS):** specifies recursive DNS servers for this
  network, for use with SLAAC-configured interfaces.

## Using NDP in Go

Now that we've learned that some of the fundamentals of NDP with IPv6, let's
examine a Go package, [`github.com/mdlayher/ndp`](https://github.com/mdlayher/ndp),
which allows NDP to be used in Go programs.

The fundamental type of the NDP package is `ndp.Conn`. An `ndp.Conn` is used to
bind an ICMPv6 connection on a network interface for the purposes of sending and
receiving NDP messages.

```go
// Select a network interface by its name to use for NDP communications.
ifi, err := net.InterfaceByName("eth0")
if err != nil {
    log.Fatalf("failed to get interface: %v", err)
}

// Set up an *ndp.Conn, bound to this interface's link-local IPv6 address.
c, ip, err := ndp.Dial(ifi, ndp.LinkLocal)
if err != nil {
    log.Fatalf("failed to dial NDP connection: %v", err)
}
// Clean up after the connection is no longer needed.
defer c.Close()

fmt.Println("ndp: bound to address:", ip)
// ndp: bound to address: fe80::76d4:35ff:fee7:cbc4
```

The typical use case of an `ndp.Conn` is to send and receive NDP messages which
implement the `ndp.Message` interface. The messages implemented in the package
mirror the name and structure of the messages defined in the NDP specification.
Within these messages, NDP options can be specified using types which implement
the `ndp.Option` interface.

As an example, let's demonstrate use of the package by sending a Neighbor
Solicitation message to discover the MAC address of an IPv6 neighbor, using its
link-local address.

When using ARP, the equivalent message would be broadcast to all machines on the
same local network. With IPv6, broadcast is completely eliminated in favor of
increased use of multicast groups. With NDP, we can use an interface's
[solicited-node multicast address](https://en.wikipedia.org/wiki/Solicited-node_multicast_address)
to greatly reduce the amount of network traffic required for this operation.

```go
// Choose a target with a known IPv6 link-local address.
target := net.ParseIP("fe80::1e1b:dff:feea:830f")

// Use target's solicited-node multicast address to request that the target
// respond with a neighbor advertisement.
snm, err := ndp.SolicitedNodeMulticast(target)
if err != nil {
    log.Fatalf("failed to determine solicited-node multicast address: %v", err)
}

// Build a neighbor solicitation message, indicate the target's link-local
// address, and also specify our source link-layer address.
m := &ndp.NeighborSolicitation{
    TargetAddress: target,
    Options: []ndp.Option{
        &ndp.LinkLayerAddress{
            Direction: ndp.Source,
            Addr:      ifi.HardwareAddr,
        },
    },
}

// Send the multicast message and wait for a response.
if err := c.WriteTo(m, nil, snm); err != nil {
    log.Fatalf("failed to write neighbor solicitation: %v", err)
}
msg, _, from, err := c.ReadFrom()
if err != nil {
    log.Fatalf("failed to read NDP message: %v", err)
}

// Expect a neighbor advertisement message with a target link-layer
// address option.
na, ok := msg.(*ndp.NeighborAdvertisement)
if !ok {
    log.Fatalf("message is not a neighbor advertisement: %T", msg)
}
if len(na.Options) != 1 {
    log.Fatal("expected one option in neighbor advertisement")
}
tll, ok := na.Options[0].(*ndp.LinkLayerAddress)
if !ok {
    log.Fatalf("option is not a link-layer address: %T", msg)
}

fmt.Printf("ndp: neighbor advertisement from %s:\n", from)
fmt.Printf("  - solicited: %t\n", na.Solicited)
fmt.Printf("  - link-layer address: %s\n", tll.Addr)

// ndp: neighbor advertisement from fe80::1e1b:dff:feea:830f:
//   - solicited: true
//   - link-layer address: 1c:1b:0d:ea:83:0f
```

As a final example, let's send a Router Solicitation to discover routers and
IPv6 prefix information on our local network. When the "autonomous address
autoconfiguration" flag is set in a Router Advertisement, IPv6-enabled
interfaces can use it to configure their own IPv6 addresses automatically;
no DHCPv6 server required!

```go
// Build a router solicitation message, specifying our source link-layer
// address so the router does not have to ask it for it explicitly.
m := &ndp.RouterSolicitation{
    Options: []ndp.Option{
        &ndp.LinkLayerAddress{
            Direction: ndp.Source,
            Addr:      ifi.HardwareAddr,
        },
    },
}

// Send to the "IPv6 link-local all routers" multicast group and wait
// for a response.
if err := c.WriteTo(m, nil, net.IPv6linklocalallrouters); err != nil {
    log.Fatalf("failed to write router solicitation: %v", err)
}
msg, _, from, err := c.ReadFrom()
if err != nil {
    log.Fatalf("failed to read NDP message: %v", err)
}

// Expect a router advertisement message.
ra, ok := msg.(*ndp.RouterAdvertisement)
if !ok {
    log.Fatalf("message is not a router advertisement: %T", msg)
}

// Iterate options and display information.
fmt.Printf("ndp: router advertisement from %s:\n", from)
for _, o := range ra.Options {
    switch o := o.(type) {
    case *ndp.PrefixInformation:
        fmt.Printf("  - prefix %q: SLAAC: %t\n", o.Prefix, o.AutonomousAddressConfiguration)
    case *ndp.LinkLayerAddress:
        fmt.Printf("  - link-layer address: %s\n", o.Addr)
    }
}

// ndp: router advertisement from fe80::618:d6ff:fea1:ceb7:
//   - prefix "2600:6c4a:787f:d200::": SLAAC: true
//   - prefix "fd00::": SLAAC: true
//   - link-layer address: 04:18:d6:a1:ce:b7
```

## Summary

The Neighbor Discovery Protocol is a crucial and fundamental part of IPv6
networking. Compared to ARP and IPv4, NDP and IPv6 offer numerous advantages:

- NDP builds on top of ICMPv6 using link-local addresses, instead of dealing
  with a totally different protocol on top of Ethernet frames.
- IPv6 makes smart use of multicast to eliminate the need for broadcast,
  preventing a lot of unnecessary network noise.
- NDP can be easily extended because new options can be added in a
  backward-compatible way.

Although your operating system will take care of NDP on your behalf for normal
applications, it can occasionally be useful to exercise the power of NDP and
other low-level networking protocols directly. If you're interested in exploring
NDP traffic on your local network, check out the `ndp` tool included in my
repository.

```text
go get -u github.com/mdlayher/ndp/...
```

Here's an example of using `ndp` to send Router Solicitations on interface
`eth0` from the interface's link-local address until a Router Advertisement is
received:

```text
$ sudo ndp -i eth0 -a linklocal rs
ndp> interface: eth0, link-layer address: 04:18:d6:a1:ce:b8, IPv6 address: fe80::618:d6ff:fea1:ceb8
ndp rs> router solicitation:
    - source link-layer address: 04:18:d6:a1:ce:b8

ndp rs> router advertisement from: fe80::201:5cff:fe69:f246:
    - hop limit:        0
    - flags:            [MO]
    - preference:       0
    - router lifetime:  2h30m0s
    - reachable time:   1h0m0s
    - retransmit timer: 0s
    - options:
        - prefix information: 2600:6c4a:7002:100::/64, flags: [], valid: 720h0m0s, preferred: 168h0m0s
```

If you'd like to make use of NDP in your own applications, I encourage you to
try out my Go package, [`github.com/mdlayher/ndp`](https://github.com/mdlayher/ndp).

It is currently used in [MetalLB](https://github.com/google/metallb) to expose
Kubernetes services over IPv6 addresses, and for internal projects at
DigitalOcean. Perhaps it could also be used as a model to implement your own
NDP library in your programming language of choice!

This blog provides a high-level overview to get readers started with NDP, but
if you'd like to learn more about some of the common use cases for NDP in
computer networks today, check out
[Jeremy Stretch's blog on the topic at PacketLife](http://packetlife.net/blog/2008/aug/28/ipv6-neighbor-discovery/).

Thank you very much for reading this post. I hope you've enjoyed it and learned
something new along the way. If you have, you may also be interested in
[some of my other posts](/blog) about using low-level networking primitives with
the Go programming language.

Finally, if you have questions or comments, feel free to reach out on
[Twitter](https://twitter.com/mdlayher) or [Gophers Slack](https://gophers.slack.com/)
(username: mdlayher).

Thanks again for your time!

## Links

- [Package/command `ndp` for Go](https://github.com/mdlayher/ndp)
- [PacketLife: IPv6 neighbor discovery](http://packetlife.net/blog/2008/aug/28/ipv6-neighbor-discovery/)

## References

- [Wikipedia: ARP](https://en.wikipedia.org/wiki/Address_Resolution_Protocol)
- [Wikipedia: NDP](https://en.wikipedia.org/wiki/Neighbor_Discovery_Protocol)
- [IETF RFC 4861](https://tools.ietf.org/html/rfc4861)
- [Wikipedia: Link-local address (IPv6)](https://en.wikipedia.org/wiki/Link-local_address#IPv6)
- [Wikipedia: ICMPv6](https://en.wikipedia.org/wiki/Internet_Control_Message_Protocol_for_IPv6)
- [Wikipedia: IPv6: SLAAC](https://en.wikipedia.org/wiki/IPv6#Stateless_address_autoconfiguration_%28SLAAC%29)
- [Wikipedia: Solicited-node multicast address](https://en.wikipedia.org/wiki/Solicited-node_multicast_address)
