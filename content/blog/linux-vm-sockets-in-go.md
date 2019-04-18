+++
title = "Linux VM sockets in Go"
date = "2017-05-08T08:00:00+00:00"
subtitle = "Leveraging Linux VM sockets to enable bidirectional, many-to-one communication between a hypervisor and its VMs, using Go."
+++

During a recent discussion with coworkers, I discovered a new Linux socket
family: VM sockets (`AF_VSOCK` address family). This new socket family enables
bi-directional, many-to-one, communication between a hypervisor and its virtual
machines, using the classic BSD sockets API.

Although VM sockets were originally introduced by VMware, they can be used with
QEMU+KVM virtual machines as well. This post will detail how VM sockets work and
how they can be used.

If you'd like to see more examples and make use of VM sockets in your own Go
applications, check out: [github.com/mdlayher/vsock](https://github.com/mdlayher/vsock).

## Introduction to VM sockets

VM sockets were added to the kernel to overcome some of the limitations that
existing communication mechanisms faced:

- Serial port communication is meant for one-to-one, not many-to-one
  communications.
- Only 512 serial ports are available (relatively low limit).

Because VM sockets do not rely on the host's networking stack at all, it is
possible to configure VMs entirely without networking: only allowing
communication using VM sockets.

## VM sockets setup

To take advantage of VM sockets (using
[virtio-vsock](http://wiki.qemu.org/Features/VirtioVsock)), the Linux kernel (on
both the hypervisor and guest) and QEMU must be fairly up-to-date. Kernel 4.8+
is required on both machines, and QEMU 2.8+ is required to execute the VM.

Once these components are in place, some setup must be done on the hypervisor
to enable VM sockets communication.

First, the necessary kernel modules must be loaded on the hypervisor (with
kernel 4.8+).

```text
hypervisor $ uname -a  
Linux hypervisor 4.8.0-39-generic #42~16.04.1-Ubuntu SMP Mon Feb 20 15:06:07 UTC 2017 x86_64 x86_64 x86_64 GNU/Linux  
hypervisor $ sudo modprobe vhost_vsock
```

Once the kernel module is loaded, two special character devices will appear on
the hypervisor.

```text
hypervisor $ ls -l /dev/vhost-vsock
crw------- 1 root root 10, 53 May  4 11:55 /dev/vhost-vsock  
hypervisor $ ls -l /dev/vsock
crw-rw-rw- 1 root root 10, 54 May  4 11:55 /dev/vsock
```

Next, QEMU must be started with a special `vhost-vsock-pci` device attached
that enables VM sockets communication within the VM. Note that each VM on a
hypervisor must have a unique "cid" (context ID). For this example, we've
chosen `guest-cid=3`.

```text
hypervisor $ sudo qemu-system-x86_64 -m 4G -hda /home/matt/ubuntuvm0.img \
             -device vhost-vsock-pci,id=vhost-vsock-pci0,guest-cid=3 -vnc :0 \
             --enable-kvm
```

Within the virtual machine, verify that the `/dev/vsock` device is available.

```text
vm $ ls -l /dev/vsock
crw-rw-rw- 1 root root 10, 55 May  4 13:21 /dev/vsock
```

## VM sockets addresses

A VM sockets address is comprised of a context ID and a port; just like an IP
address and TCP/UDP port.

The context ID (CID) is analogous to an IP address, and is represented using an
unsigned 32-bit integer. It identifies a given machine as either a hypervisor
or a virtual machine. Several addresses are reserved, including `0`, `1`, and
the maximum value for a 32-bit integer: `0xffffffff`. The hypervisor is always
assigned a CID of `2`, and VMs can be assigned any CID between `3` and
`0xffffffff — 1`.

A port is analogous to a typical TCP or UDP port, and is represented using an
unsigned 32-bit integer. Many different services can run on the same host by
binding to different ports, and each port can serve multiple connections
concurrently. As with IP ports, ports in the range `0-1023` are considered
"privileged", and only `root` or a user with `CAP_NET_ADMIN` may bind to
these ports.

## VM sockets API

Now that we are familiar with some of the basics of VM sockets, let's dive
into the API. As with other socket types on Linux, the BSD sockets API is
used when configuring VM sockets. It appears that VM sockets can be used in
both connection-oriented (like TCP) and connection-less (like UDP) modes, but
this post will only cover the connection-oriented variant.

All pseudo-code in this post will make use of Go's [`golang.org/x/sys/unix`](https://godoc.org/golang.org/x/sys/unix)
package.

If you have experience creating TCP sockets using system calls, this process
should seem quite familiar. First, let's start up a VM sockets server on the
hypervisor.

```go
// Retrieve host's context ID from /dev/vsock. More on this later.  
cid := localContextID()

// Establish a connection-oriented VM socket.  
socket, err := unix.Socket(unix.AF_VSOCK, unix.SOCK_STREAM, 0)  
if err != nil {  
    return err  
}

// Bind socket to local context ID, port 1024.  
sockaddr := &unix.SockaddrVM{  
    CID:  cid,  
    Port: 1024,  
}
if err := unix.Bind(socket, sockaddr); err != nil {  
    return err  
}

// Listen for up to 32 incoming connections.  
fd, err := unix.Listen(socket, 32)  
if err != nil {  
    return err  
}

// Use fd to read and write data to and from a VM.
```

Next, we can dial out to the server running on the hypervisor, from a client
running in the VM.

```go
// Establish a connection-oriented VM socket.  
socket, err := unix.Socket(unix.AF_VSOCK, unix.SOCK_STREAM, 0)  
if err != nil {  
    return err  
}

// Connect socket to hypervisor context ID, port 1024.  
sockaddr := &unix.SockaddrVM{  
    CID:  2,  
    Port: 1024,  
}  
if err := unix.Connect(socket, sockaddr); err != nil {  
   return err  
}

// Use fd to read and write data to and from the hypervisor.
```

As you can see, VM sockets can more or less be used as a drop-in replacement
for typical TCP sockets.

## Retrieving the local context ID

When working with VM sockets, it can be useful to retrieve the local context ID
of a given machine. This can be done by performing an `ioctl()` system call on
`/dev/vsock`.

```go
// Open /dev/vsock to perform ioctl().  
f, err := os.Open("/dev/vsock")  
if err != nil {  
    return err  
}  
defer f.Close()

// Ask kernel to deference a pointer to cid and place the local  
// CID for this host in the uint32 value "cid".  
var cid uint32  
err := ioctl(
    f.Fd(),
    unix.IOCTL_VM_SOCKETS_GET_LOCAL_CID,
    uintptr(unsafe.Pointer(&cid)),
)
if err != nil {  
    return err  
}
```

Because of the very versatile and somewhat dangerous nature of the `ioctl()`
system call, it is not implemented directly in `x/sys/unix`. You can see how
I've implemented it in package `vsock` [here](https://github.com/mdlayher/vsock/blob/master/ioctl_linux.go).

## Package vsock

To simplify the VM sockets setup process and enable code reuse, I have created
a VM sockets package for Go: [github.com/mdlayher/vsock](https://github.com/mdlayher/vsock).

Using package `vsock`, one can build client/server applications in Go using VM
sockets in a rather straightforward and familiar way: using the `net.Listener`
and `net.Conn` interfaces.

As an example, let's create an "echo" service using VM sockets. A server listens
for incoming connections, and when a message is received from a client, it is
echoed back to the client.

Here's the code for the server:

```go
// Listen for VM sockets connections on port 1024.  
l, err := vsock.Listen(1024)  
if err != nil {  
    return err  
}  
defer l.Close()

// Accept a single connection.  
c, err := l.Accept()  
if err != nil {  
    return err  
}  
defer c.Close()

// Echo all data from the client back to the client.  
if _, err := io.Copy(c, c); err != nil {  
    return err  
}
```

The code for the client is succinct as well:

```go
// Dial a VM sockets connection to a process on the hypervisor  
// bound to port 1024.  
c, err := vsock.Dial(vsock.Host, 1024)  
if err != nil {  
    return err  
}  
defer c.Close()

// Send a brief message to the hypervisor.  
if _, err := c.Write([]byte("hello world")); err != nil {  
    return err  
}

// Read back the echoed response from the hypervisor.  
b := make([]byte, 16)  
n, err := c.Read(b)  
if err != nil {  
    return err  
}

fmt.Println(string(b[:n]))
```

## Summary

VM sockets are a very interesting new communication mechanism, but it may take
some time for many production environments to deploy new enough versions of the
Linux kernel and QEMU to take advantage of them.

Once widely deployed, they could become quite useful for offering additional
services on Infrastructure-as-a-Service platforms. Guest agents running inside
the VM could leverage services provided by the hypervisor in new and interesting
ways. The possibilities are limitless!

If you'd like a more in-depth look at VM sockets, I recommend
[this excellent presentation](https://vmsplice.net/~stefan/stefanha-kvm-forum-2015.pdf)
by Stefan Hajnoczi. You may also be interested in Stefan's
[proposed additions to the virtio specification](https://stefanha.github.io/virtio/#x1-2830007)
for virtio socket devices. Finally, I'd like to thank Stefan for personally
answering several of my questions about VM sockets terminology and architecture.

Thank you for reading, and I hope you've learned something new from this post.
I encourage you to set up VM sockets in a development environment, and to try
out package [`vsock`](https://github.com/mdlayher/vsock) as well!

If you enjoyed this post, you may also be interested in my series about using
[Netlink sockets in Go](../linux-netlink-and-go-part-1-netlink)! Thank you for
your time.

## References

- [Features/Virtio-Vsock - QEMU](http://wiki.qemu.org/Features/VirtioVsock)
- [virtio-vsock: Zero-configuration host/guest communication](https://vmsplice.net/~stefan/stefanha-kvm-forum-2015.pdf)
