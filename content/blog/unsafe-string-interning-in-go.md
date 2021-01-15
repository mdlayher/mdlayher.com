+++
date = "2020-12-30T00:00:00+00:00"
title = "Unsafe string interning in Go"
subtitle = "I discuss the https://go4.org/intern package and the unsafe techniques it uses to implement string interning in Go."
+++

> Note: this project continues to evolve and some theoretical concerns about the
> future safety of this approach have been identified and documented here:
> https://github.com/go4org/intern/issues/13.
>
> If you're using this post as a reference, please look at the latest version of
> package `intern` instead.

For a few months, I've been collaborating on the
[`inet.af/netaddr`](https://inet.af/netaddr) package for Go which introduces a
new `netaddr.IP` type as an alternative to the standard library's
[`net.IP`](https://golang.org/pkg/net/#IP) type. There are [a variety of reasons
for this](https://pkg.go.dev/inet.af/netaddr#readme-motivation), but those are
not the focus of today's blog.

The folks over at [Tailscale](https://tailscale.com/) have been working on
improving the performance of the package in various ways, including shrinking
the size of types and trying out faster internal IP representations.
Specifically, one topic that came up was [how to shrink `netaddr.IP` from 32
bytes to 24 bytes](https://github.com/inetaf/netaddr/pull/57), the same size
as a Go slice header.

Because a `netaddr.IP` can also have an associated zone for IPv6 link-local
addresses, the type previously contained a Go string, which occupies 16 bytes.
This is a good use case for [string
interning](https://en.wikipedia.org/wiki/String_interning), a technique which
can be used to reduce the number of string allocations. It is highly likely that
a small number of zone identifiers will be used repeatedly throughout a program,
so string interning seems like a great fit for this problem.

In addition, another need for `netaddr.IP` is to guarantee that identical input
strings will also have identical backing data pointers in the intern pool. This
ensures that the input strings are usable internally as map keys, can occupy
less space in the pool, and can be used to occupy only one word of memory within
the type rather than two, as a Go string header would.

The result of this work is the package
[`go4.org/intern`](https://go4.org/intern) which uses some pretty neat `unsafe`
tricks to implement efficient string interning using weak references and Go
finalizers. We'll start by showing off the safe implementation and gradually
introduce the concepts needed to understand the unsafe one as well.

All credit for the ideas and implementation of `go4.org/intern` goes to [Dave
Anderson](https://twitter.com/dave_universetf), [Josh Bleecher
Snyder](https://twitter.com/commaok), and [Brad
Fitzpatrick](https://twitter.com/bradfitz). I simply watched the discussions
unfold on the issue tracker and thought that the code and the ideas behind it
were too interesting to not write up and share!

## Safe string interning in Go

The basic idea of string interning is to retain a global pool of strings which
can be created or fetched on the fly, so that each string is allocated exactly
once for values which are identical. It's worth noting that since our code will
accept empty `interface{}`, you could use it with any type which can be used as
a map key. This blog will focus on strings, but the same API could be used for
comparing the equality of a large struct type as well.

In its most basic form, you could implement `go4.org/intern` (with a caveat!) by
writing the following:

```go
package intern

// A Value pointer is the handle to an underlying comparable value.
// See func Get for how Value pointers may be used.
type Value struct {
	// Enforce that Values cannot be used directly for equality comparisons.
	// We'll explain why a bit later.
	_      [0]func()
	cmpVal interface{}
}

// Get returns the comparable value passed to the Get func
// that returned v.
func (v *Value) Get() interface{} { return v.cmpVal }

// Our pool of interned values and a lock to serialize access.
var (
	mu      sync.Mutex
	val = map[interface{}]*Value{}
)

// Get returns a pointer representing the comparable value cmpVal.
//
// The returned pointer will be the same for Get(v) and Get(v2)
// if and only if v == v2, and can be used as a map key.
//
// Note that Get returns a *Value so we only return one word of data
// to the caller, despite potentially storing a large amount of data
// within the Value itself.
func Get(cmpVal interface{}) *Value {
	mu.Lock()
	defer mu.Unlock()

	v := val[cmpVal]
	if v != nil {
		// Value is already interned in the pool.
		return v
	}

	// Value must be created now and then interned.
	v = &Value{cmpVal: cmpVal}
	val[cmpVal] = v
	return v
}
```

Usage of the package is straightforward at this point:

```go
func TestBasics(t *testing.T) {
	foo := intern.Get("foo")
	foo2 := intern.Get("foo")

	if foo.Get() != foo2.Get() {
		t.Error("foo values differ")
	}
	if foo.Get() != "foo" {
		t.Error("foo.Get not foo")
	}
	if foo != foo2 {
		t.Error("foo pointers differ")
	}
}
```

The global pool in the library stored the string `"foo"` on the first call to
`intern.Get("foo")` and the same `*intern.Value` was returned and stored in both
`foo` and `foo2`.

However there is a caveat to this approach: the pool can never shrink. If we
keep adding more distinct values to the pool, it will continue to grow in size
and allocate more memory until our program eventually runs out of memory and
crashes. Even if we don't add more values, we've still stranded some of our
memory for the entire duration of the program's execution.

In fact, this is the tradeoff made by `go4.org/intern` for the "safe" operating
mode of the package. If you run your program with the environment variable
`GO4_INTERN_SAFE_BUT_LEAKY=true` then no unsafe tricks will be used, and the
intern pool will continue to grow as more items are added. For some programs,
this may be an acceptable tradeoff, but there's much more to be discussed!

## Breaking the unsafe rules

In order to make use of string interning while also cooperating with Go's
garbage collector to remove and free unused values from the pool, it is
necessary to bend the package `unsafe` rules. This is what inspired me to author
this blog: I've written a lot of unsafe code in the past but have never
intentionally stepped outside the guard rails that the [`unsafe.Pointer` rules
provide for you](https://golang.org/pkg/unsafe/#Pointer).

Specifically, we have to violate rule number 3:

> (3) Conversion of a Pointer to a uintptr and back, with arithmetic.
>
> ... Note that both [uintptr and unsafe.Pointer] conversions must appear in the
> same expression, with only the intervening arithmetic between them:

```go
// INVALID: uintptr cannot be stored in variable
// before conversion back to Pointer.
u := uintptr(p)
p = unsafe.Pointer(u + offset)
```

In a hypothetical future version of Go, it is possible that the value stored at
the address pointed at by the `p` in the Go heap could be moved in between the
conversion and assignment `u := uintptr(p)` and the pointer arithmetic `p =
unsafe.Pointer(u + offset)`.

Because we are relying on the invariant that the Go garbage collector (as of Go
1.15) does not move data in the heap, [Brad
Fitzpatrick](https://twitter.com/bradfitz) came up with a clever pseudo-package
which users can underscore import if they intend to violate this particular
package `unsafe` rule:
[`go4.org/unsafe/assume-no-moving-gc`](https://pkg.go.dev/go4.org/unsafe/assume-no-moving-gc).
If a program or its dependencies import this package and a future version of Go
uses a moving garbage collector which may rearrange items in the heap, the
package can be updated to fail compilation when the program's behavior is now
invalid.

Technically, Go _does_ have a moving garbage collector for its stacks as of Go
1.15. See [Matthew Dempsky's issue for more
details](https://github.com/go4org/unsafe-assume-no-moving-gc/issues/3). Because
`go4.org/intern` stores values in a global map, the values will never be stack
allocated and thus this is not a problem for this particular use case.

Now that we've set the terms of engagement, it's time to dive in to the unsafe
implementation.

## Unsafe string interning in Go

We'll be implementing the package `intern` code with the same API as before, but
with our newfound `unsafe` tricks and some [Go runtime
finalizers](https://golang.org/pkg/runtime/#SetFinalizer) as well.

```go
package intern

// A Value pointer is the handle to an underlying comparable value.
// See func Get for how Value pointers may be used.
type Value struct {
	// Enforce that Values cannot be used directly for equality comparisons.
	// This is necessary because of the resurrected boolean below.
	_      [0]func()
	cmpVal interface{}
	// resurrected is guarded by mu (for all instances of Value).
	// It is set true whenever v is synthesized from a uintptr.
	resurrected bool
}

// Get returns the comparable value passed to the Get func
// that returned v.
func (v *Value) Get() interface{} { return v.cmpVal }

var (
	// mu guards val, a weak reference map of *Value by underlying value.
	// It also guards the resurrected field of all *Values.
	mu   sync.Mutex
	val  = map[interface{}]uintptr{} // to uintptr(*Value)
)

// We play unsafe games that violate Go's rules (and assume a non-moving
// collector). So we quiet Go here.
//go:nocheckptr

// Get returns a pointer representing the comparable value cmpVal.
//
// The returned pointer will be the same for Get(v) and Get(v2)
// if and only if v == v2, and can be used as a map key.
//
// Note that Get returns a *Value so we only return one word of data
// to the caller, despite potentially storing a large amount of data
// within the Value itself.
func Get(cmpVal interface{}) *Value {
	mu.Lock()
	defer mu.Unlock()

	var v *Value
	if addr, ok := val[cmpVal]; ok {
		// Value is already interned in the pool. In case a finalizer runs in
		// the near future, we must "resurrect" this value to bring it back
		// to the active part of its lifecycle.
		v = (*Value)((unsafe.Pointer)(addr))
		v.resurrected = true
	}
	if v != nil {
		return v
	}

	// Value must be created now and then interned. We store a uintptr to
	// maintain a weak reference so that the value can eventually be garbage
	// collected if it isn't used for a period of time.
	v = &Value{cmpVal: cmpVal}
	val[cmpVal] = uintptr(unsafe.Pointer(v))
	runtime.SetFinalizer(v, finalize)

	return v
}

// finalize is a Go finalizer for *Value.
func finalize(v *Value) {
	mu.Lock()
	defer mu.Unlock()
	if v.resurrected {
		// We lost the race. Somebody resurrected it while we were about to
		// finalize it. Try again next round.
		v.resurrected = false
		runtime.SetFinalizer(v, finalize)
		return
	}
	delete(val, v.cmpVal)
}
```

The code comments above discuss many of the ideas at work, but let's break
down `Get` and `finalize` specifically to discuss how these interact with each
other and the Go runtime.

Starting with `Get`, we acquire the mutex around the map of interned values,
which now stores the values as [weak references using
`uintptr`](https://en.wikipedia.org/wiki/Weak_reference) rather than a concrete
`*Value` type. This ensures that the values are not protected from garbage
collection, and thus we must manually manage their lifecycles by using
finalizers and the `v.resurrected` boolean field.

First we check if the value is already present in the pool. If it is, we convert
the weak reference back to a `*Value` by using `unsafe.Pointer` conversion, and
mark the value as resurrected so that any finalizers which run concurrently will
know that this value cannot be removed from the pool.

```go
	var v *Value
	if addr, ok := val[cmpVal]; ok {
		// Value is already interned in the pool. In case a finalizer runs in
		// the near future, we must "resurrect" this value to bring it back
		// to the active part of its lifecycle.
		v = (*Value)((unsafe.Pointer)(addr))
		v.resurrected = true
	}
	if v != nil {
		return v
	}
```

If the value did not already exist in the pool, we can create it now and intern
it there by adding it to the map of weak references. Note that the
`v.resurrected` field defaults to `false`. When the `*Value` is first created,
it is a strong reference which can be tracked by Go's garbage collector. The
resurrection trick becomes necessary because we must materialize a strong
reference from a weak reference when `Get` is called and the value is already
interned in the pool. If `Get` is never called, the created `*Value` can be
deleted from the map and freed by the garbage collector.

```go
	// Value must be created now and then interned. We store a uintptr to
	// maintain a weak reference so that the value can eventually be garbage
	// collected if it isn't used for a period of time.
	v = &Value{cmpVal: cmpVal}
	val[cmpVal] = uintptr(unsafe.Pointer(v))
	runtime.SetFinalizer(v, finalize)
```

Finalizers in Go come with [many
caveats](https://golang.org/pkg/runtime/#SetFinalizer) but we make use of them
here in order to ensure that weak references in the map can be deleted when they
are no longer needed, and thus the underlying `*Value` can be garbage collected
from the heap at some point in the future.

The `resurrected` field exists in order to avoid a data race. The original
implementation ran into a subtle interaction between Go's garbage collector and
runtime finalizers. Consider the following scenario [laid out by Dave
Anderson](https://github.com/go4org/intern/issues/2#issuecomment-749925320):

- GC runs mark phase. Value is dangling, so gets earmarked for collection (w/ finalizer run).
- Get materializes a new pointer to Value. GC won't notice this until the next mark phase.
- Get clears finalizer.
- Concurrent GC gets around to processing that earmarked Value, finds it has no finalizer. Great! free().
- Get continues to operate on its materialized pointer, which now points to freed memory.

To avoid this race, we must verify the `resurrected` field and set another
finalizer if another caller managed to fetch this `*Value` using `Get`
concurrently, or if the concurrent garbage collector was in the middle of its
mark and sweep process. If the value was not resurrected before the finalizer
runs, we are able to finally free it from the weak reference pool.

```go
// finalize is a Go finalizer for *Value.
func finalize(v *Value) {
	mu.Lock()
	defer mu.Unlock()

	if v.resurrected {
		// We lost the race. Somebody resurrected it while we were about to
		// finalize it. Try again next round.
		v.resurrected = false
		runtime.SetFinalizer(v, finalize)
		return
	}
	delete(val, v.cmpVal)
}
```

Because of the interactions here between weak references, finalizers, and
possible concurrent calls to `Get`, it may take up to three garbage collection
cycles to remove a `*Value` from the pool and free it.

1. the finalizer runs and the `resurrected` sentinel is cleared
1. the finalizer runs again and deletes the value from the weak reference pool
1. the garbage collector is able to free the associated `*Value` in the heap

There's one more subtle interaction that is worth mentioning. [Josh Bleecher
Snyder points out:](https://github.com/mdlayher/mdlayher.com/pull/8/files#r549506397)

> It's not exactly guaranteed that two identical values will have the same
> *Value. It's only guaranteed that you won't be able to observe two different
> values at the same time.

You must be careful to avoid hiding `*intern.Value`s from the Go garbage
collector via unsafe tricks or you will violate the package's guarantee that
the same pointer points to an identical value. Don't persist these values to
a database or keep dangling `uintptr` values around!

And with that, we've successfully implemented unsafe string interning in Go
using weak references and finalizers! To see the completed implementation and
for more technical details and background, [check out the code in the
`go4.org/intern`](https://github.com/go4org/intern/blob/main/intern.go)
repository.

## Conclusion

The unsafe techniques used in [`go4.org/intern`](https://go4.org/intern) are
incredibly subtle and have some seriously sharp edges. There have been several
different iterations of this package written by some of the most experienced Go
programmers on the planet that _still_ managed to have subtle defects and data
races.

You can follow the saga by checking out some of the following:

- [initial
  commit](https://github.com/go4org/intern/commit/bb95bb3e874c5b9a81af3278efba53053bc57775)
- [fatal error: found pointer to free
  object](https://github.com/go4org/intern/issues/2)
- [Fix runtime crash](https://github.com/go4org/intern/pull/4)
- [Rename, change "refs" field from int to "resurrected" bool](https://github.com/go4org/intern/commit/ef8cbcb8edd7fe8843ea3a1dd5d92b6791d778d8)

I strongly recommend giving this package a try if you need the guarantees it
provides rather than trying to roll your own!

Special thanks again to [Dave Anderson](https://twitter.com/dave_universetf),
[Josh Bleecher Snyder](https://twitter.com/commaok), and [Brad
Fitzpatrick](https://twitter.com/bradfitz) for all of their work and discussions
that ultimately led to the creation of this package. And another round of thanks
for their review of the rough drafts for this blog!

If you have questions or comments, feel free to [reach out on
Twitter](https://twitter.com/mdlayher) or [@mdlayher on Gophers
Slack](https://invite.slack.golangbridge.org/). I wrote this blog during a
couple of live streams (hello Twitch chat!) and [regularly stream Go content on
Twitch](https://twitch.tv/mdlayher), so feel free to stop by and say hi! Thanks
for reading!
