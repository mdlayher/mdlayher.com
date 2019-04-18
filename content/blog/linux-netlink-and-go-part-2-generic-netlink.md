+++
date = "2017-02-28T00:00:00+00:00"
title = "Linux, Netlink, and Go - Part 2: generic netlink"
subtitle = "An introduction to generic netlink: an extensible netlink family. This post also describes how to leverage generic netlink in Go."
+++

In [Part 1 of this series](../linux-netlink-and-go-part-1-netlink), I
described some of the fundamental concepts of netlink: sockets, messages, and
attributes. It is assumed that readers are already familiar with netlink from
the previous post in this series.

In this post, I will dive into **generic netlink**, a specialized netlink family
designed to be more extensible than a typical netlink family.

The pseudo-code in this series will use Go's [`golang.org/x/sys/unix`](https://godoc.org/golang.org/x/sys/unix)
package and types from my `netlink`, `genetlink`, and `wifi` packages. The
series is broken up as follows:

- [Part 1: netlink](../linux-netlink-and-go-part-1-netlink): an introduction
  to netlink.
- [Part 2: generic netlink](.): an introduction to generic netlink, a netlink
  family meant to simplify creation of new families.
- [Part 3: packages netlink, genetlink, and wifi](../linux-netlink-and-go-part-3-packages-netlink-genetlink-and-wifi):
  using Go to drive interactions with netlink, generic netlink, and nl80211.

## What is generic netlink?

Generic netlink was designed to allow kernel modules to easily communicate
with userspace applications using netlink. Instead of creating a new top level
netlink family for each module, generic netlink provides a simplified interface
to enable modules to plug into the generic netlink bus.

## Generic netlink messages

Generic netlink makes use of a message with a small payload. This message
occupies the first four bytes of a netlink message payload, and looks like so:

- **Command** (8 bits): specifies which command to issue to a generic netlink
  family.
- **Version** (8 bits): the version of a command to issue to generic netlink.

2 bytes of padding immediately follow these fields. Remember, netlink makes use
of 4 byte boundaries! After the padding, a generic netlink message payload is
present. This payload may contain data (such as netlink attributes) as
parameters for a command, or data in response to a command.

## Generic netlink families

Not to be confused with netlink's families, generic netlink also has a concept
of families: kernel modules which register with the generic netlink bus for
communication. Some examples of these may include:

- **nlctrl**: the generic netlink controller, used to determine which generic
  netlink families are available. Present on all systems where generic netlink
  is available.
- **TASKSTATS**: provides per-task and per-process statistics from the kernel
  to userspace.
- **nl80211**: provides access to IEEE 802.11 WiFi device statistics and
  interactions.

## nlctrl: the generic netlink controller

In order to discover which generic netlink families are available on a given
machine, a request can be sent to the generic netlink controller. The controller
is a special family which is present on all machines where generic netlink
is available.

To send a message to the controller, its family ID (always `0x10`) is used in
the Type field of the outer netlink header. The payload of the netlink message
contains the generic netlink header, which specifies a command and its version.
Finally, parameters can be passed as netlink attributes in the body of the
generic netlink message.

When you put it all together, the command "controller: list all generic netlink
families" looks something like this:

```go
msg := netlink.Message{
    Header: netlink.Header{
        // Specify nlctrl's type (0x10) to communicate with it.
        Type: genetlink.Controller,
        Flags: netlink.Request | netlink.Dump,
        // Some fields omitted for brevity.
    },
    // The generic netlink header and data are wrapped in a
    // netlink message, marshaled into byte form.
    Data: marshal(genetlink.Message{
        Header: genetlink.Header{
            Command: ctrlCommandGetFamily,
            Version: ctrlVersion,
        },
    }),
}
```

Requesting information about a single family from the controller is slightly
different. The "dump" flag is omitted from the netlink header, and the generic
netlink message payload contains attributes which specify the name of a
specific family:

```go
// Used in Data field of genetlink.Message.
b := netlink.MarshalAttributes([]netlink.Attribute{{
    Type: attrFamilyName,
    // Null-terminated string in byte form.
    Data: nlenc.Bytes("nl80211"),
}})
```

Many of these constants are taken from the [generic netlink kernel headers](https://elixir.bootlin.com/linux/v5.0.8/source/include/linux/genetlink.h).
To see them in action in Go source code, you may also reference
package [`genetlink`](https://github.com/mdlayher/netlink/tree/master/genetlink).

## Generic netlink family attributes

Family information is returned in a generic netlink message payload as a series
of netlink attributes. These attributes include information about a generic
netlink family, such as:

- **ID** (16 bits): unique identifier for family. **Note**: ID may change between
  reboots or if certain kernel modules are loaded or unloaded. Always perform a
  lookup by name to retrieve a family's ID!
- **Name** (null-terminated string): human-readable name for the family, like
  "nlctrl", "TASKSTATS", or "nl80211".
- **Version** (32 bits): version of generic netlink family. Oddly, this field is
  32 bits while the version field in the generic netlink header is 8 bits. I
  have never seen this value occupy more than 8 bits.
- **Multicast groups** (nested attributes): netlink attribute "array" with
  attribute type incremented by one for each element. Contains additional nested
  attributes with multicast group name (null-terminated string) and ID (32 bits).

Several other attributes exist as well, but as I have not worked with them, I
cannot explain their use.

## Summary

Thanks to the foundation provided by netlink, generic netlink provides a simple
and elegant mechanism for extending netlink. There are many available generic
netlink families, and the list may differ greatly from system to system.

To discover which generic netlink families are available on your machine, try
`genl-ctrl-list`. Specifying `-d` provides a great amount of detail about
each family.

```plaintext
$ genl-ctrl-list
0x0010 nlctrl version 2
0x0011 VFS_DQUOT version 1
0x0013 NLBL_MGMT version 3
0x0014 NLBL_CIPSOv4 version 3
0x0015 NLBL_UNLBL version 3
0x0016 acpi_event version 1
0x0017 thermal_event version 1
0x0018 tcp_metrics version 1
0x0019 TASKSTATS version 1
0x001a nl80211 version 1
```

The final part of this series will detail high-level usage of netlink, generic
netlink, and nl80211, using Go packages `netlink`, `genetlink`, and `wifi`.

Thanks again for reading! If you have questions or comments, feel free to
reach out via [Twitter](https://twitter.com/mdlayher), or [Gophers Slack](https://invite.slack.golangbridge.org/)
(username: mdlayher).

## References

- [Linux Foundation Wiki: generic_netlink_howto](https://wiki.linuxfoundation.org/networking/generic_netlink_howto)
- A lot of time spent running `iw` commands with the `NLCB=debug` environment
  variable. `nlmon` virtual interfaces are also quite useful for capturing
  netlink traffic.
