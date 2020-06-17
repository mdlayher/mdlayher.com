+++
date = "2020-06-17T00:00:00+00:00"
title = "Go generics draft design: building a hashtable"
subtitle = "I describe my experiences porting a toy hashtable design to the June 2020 Go generics draft design."
+++

In 2018, I [implemented a toy hashtable in Go](https://github.com/mdlayher/misc/blob/master/go/algorithms/hashtable/hashtable.go)
as a quick refresher on how data types such as Go's map work under the hood.
This implementation works exclusively with string keys mapped to string values.

Two years later, in June 2020, the Go team released a blog post entitled
[The Next Step for Generics](https://blog.golang.org/generics-next-step) which
provides an updated generics draft design based on extending Go's existing
interfaces, rather than adding new concepts such as "contracts". If you haven't
yet I [highly recommend at least browsing the new design draft document](https://go.googlesource.com/proposal/+/refs/heads/master/design/go2draft-type-parameters.md).
I am not an expert and can only speak from my limited experience and time with
the design.

This blog will describe the lessons I learned
[porting my toy hashtable to the new generics draft design](https://go2goplay.golang.org/p/2pcBZTQdh3u).
If you'd like to skip the introduction and check out the generic code,
[feel free to jump to "A generic hashtable"](#a-generic-hashtable).

## A non-generic hashtable

My initial design from 2018 can only work string keys and string values.

The `Table` type is the basis of the package. It stores key/value string pairs
using a slice internally, where the number of hashtable buckets within the
slice is determined by an integer `m`:

- A **smaller** `m` means less buckets will be created, but each key stored in the
  `Table` has a higher likelihood of having to share a bucket with other
  keys, thus **slowing down lookups**
- A **larger** `m` means more buckets will be created, so each key stored in the
  `Table` has a lower likelihood of having to share a bucket with other keys,
  thus **speeding up lookups**

The `kv` type is a small helper to concisely store a key/value string pair.

```go
// Package hashtable implements a basic hashtable for string key/value pairs.
package hashtable

// A Table is a basic hashtable.
type Table struct {
	m     int
	table [][]kv
}

// A kv stores key/value data in a Table.
type kv struct {
	Key, Value string
}

// New creates a Table with m internal buckets.
func New(m int) *Table {
	return &Table{
		m:     m,
		table: make([][]kv, m),
	}
}
```

This hashtable supports two operations:

- `Get`: determines if a key is present in the hashtable, returning the value
  (if found) and a boolean which indicates if the value exists
- `Insert`: inserts a new key/value pair into the hashtable, overwriting any
  previous value for the same key

Both of these operations require a hashing function which can take an input string
and return an integer indicating the bucket where a key's value may live.

```go
// hash picks a hashtable index to use to store a string with key s.
func (t *Table) hash(s string) int {
	h := fnv.New32()
	h.Write([]byte(s))
	return int(h.Sum32()) % t.m
}
```

I chose `hash/fnv32` as a simple, non-cryptographic hash function which returns
an integer. By then computing the modulus operation `hash % t.m`, we can ensure
the resulting integer returns the index of one of our hashtable buckets.

First, here's the code for `Get`:

```go
// Get determines if key is present in the hashtable, returning its value and
// whether or not the key was found.
func (t *Table) Get(key string) (string, bool) {
    // Hash key to determine which bucket this key's value belongs in.
	i := t.hash(key)

	for j, kv := range t.table[i] {
		if key == kv.Key {
            // Found a match, return it!
			return t.table[i][j].Value, true
		}
	}

    // No match.
	return "", false
}
```

The implementation of `Table.Get` hashes the input key to determine which bucket
is used to store the key's values. Once the bucket is determined, it iterates
through all of the key/value pairs in that bucket:

- if the input key matches a key in that bucket, return the bucket's value
  and boolean true
- if no match, return an empty string and boolean false

Next, let's examine `Insert`:

```go
// Insert inserts a new key/value pair into the Table.
func (t *Table) Insert(key, value string) {
	i := t.hash(key)

	for j, kv := range t.table[i] {
		if key == kv.Key {
			// Overwrite previous value for the same key.
			t.table[i][j].Value = value
			return
		}
	}

	// Add a new value to the table.
	t.table[i] = append(t.table[i], kv{
		Key:   key,
		Value: value,
	})
}
```

`Table.Insert` must also hash the input key to determine which bucket should
be used to insert the key/value pair. When iterating through the key/value pairs
in a bucket, we may discover a matching key already exists:

- if the input key matches a key in that bucket, overwrite the key's value with
  the input value
- if no match, append a new entry to the bucket's key/value pair slice.

That's it! We've created a very basic hashtable which can be used to handle
key/value string pairs.

```go
// 8 buckets ought to be plenty.
t := hashtable.New(8)
t.Insert("foo", "bar")
t.Insert("baz", "qux")

v, ok := t.Get("foo")
fmt.Printf("t.Get(%q) = (%q, %t)", "foo", v, ok)
// t.Get("foo") = ("bar", true)
```

Let's port this existing code to the new Go generics draft design!

## A generic hashtable

Our goal is to take the existing hashtable code and make it work with arbitrary
key/value pair types. But we do have one constraint: the keys in our hashtable
must match the [predeclared type constraint `comparable`](https://go.googlesource.com/proposal/+/refs/heads/master/design/go2draft-type-parameters.md#comparable-types-in-constraints),
so we can check for equality.

For my design, I decided to enforce that both key and value types are comparable,
so I could build a simple demo using two hashtables as an index and reverse
index with flipped key/value types.

```go
// Package hashtable implements a basic hashtable for generic key/value pairs.
package hashtable

// A Table is a basic generic hashtable.
type Table(type K, V comparable) struct {
    // hash is a function which can hash a key of type K with t.m.
    hash func(key K, m int) int

	m     int
	table [][]kv
}

// A kv stores generic key/value data in a Table.
type kv(type K, V comparable) struct {
	Key   K
	Value V
}

// New creates a table with m internal buckets which uses the specified hash
// function for an input type K.
func New(type K, V comparable)(m int, hash func(K, int) int) *Table(K, V) {
	return &Table(K, V){
		hash:  hash,
        m:     m,
        // Note the parentheses around "kv(K, V)"; these are required!
		table: make([][](kv(K, V)), m),
	}
}
```

The new type parameter lists are required wherever generic types are needed,
thus each of these top-level types and functions must have the type parameter
list for `K` and `V`, both of which must also be `comparable`.

There were a couple of tricky things I learned while writing this code:

- Note that a hash function `func(K, int) int` is now a second parameter passed
  to `New`. This is necessary because we have to know how to hash any given
  generic type. I could have created a new interface with a `Hash() int`
  constraint or similar, but I wanted my hashtable to work with builtin Go
  types such as `string` and `int`, which you cannot define methods on.
- It took me a little bit of time to figure out the proper parentheses usage for
  the `make()` call when creating `Table.table`. My initial attempt used
  `make([][]kv(K, V))` which won't work with the added type parameters.

It's time to implement `Get`:

```go
// Get determines if key is present in the hashtable, returning its value and
// whether or not the key was found.
func (t *Table(K, V)) Get(key K) (V, bool) {
    // Hash key to determine which bucket this key's value belongs in.
    // Pass t.m so t.hash can perform the necessary operation "hash % t.m".
    i := t.hash(key, t.m)

    for j, kv := range t.table[i] {
        if key == kv.Key {
            // Found a match, return it!
            return t.table[i][j].Value, true
        }
    }

    // No match. The easiest way to return the zero-value for a generic type
    // is to declare a temporary variable as follows.
    var zero V
    return zero, false
}
```

A method defined on a generic type must also have associated generic type
parameters declared in its receiver. `Get` can now accept any type `K` and
return any type `V` along with `bool` to indicate whether or not the value was
found.

Aside from the modified method receiver and a few `K` and `V` types, this looks
pretty much like typical Go code, which is great!

The one slightly tricky issue here is dealing with [the zero value of a generic type](https://go.googlesource.com/proposal/+/refs/heads/master/design/go2draft-type-parameters.md#the-zero-value).
The linked issue suggests doing as we've done here by declaring `var zero V`,
but perhaps in the future there could be an easier option for doing this. I
personally would love to see `return _, false` or similar as an option for
both generic and non-generic Go.

Let's move on to `Insert`:

```go
// Insert inserts a new key/value pair into the Table.
func (t *Table(K, V)) Insert(key K, value V) {
	i := t.hash(key, t.m)

	for j, kv := range t.table[i] {
		if key == kv.Key {
            // Overwrite previous value for the same key.
			t.table[i][j].Value = value
			return
		}
	}

	// Add a new value to the table.
	t.table[i] = append(t.table[i], kv(K, V){
		Key:   key,
		Value: value,
	})
}
```

Very few modifications are necessary to make this code generic:

- the method receiver is now type `*Table(K, V)` instead of `*Table`
- the input parameters are now `(key K, value V)` instead of `(key, value string)`
- the `kv{}` struct literal must now be declared as `kv(K, V){}`

That's all it takes! We now have a generic hashtable type which can accept any
keys and values which implement the `comparable` type constraint.

## Generic hashtable usage

To test this code, I decided to create two parallel hashtables which act as an
index and a reverse index between string and integer types:

```go
t1 := hashtable.New(string, int)(8, func(key string, m int) int {
	h := fnv.New32()
	h.Write([]byte(key))
	return int(h.Sum32()) % m
})

t2 := hashtable.New(int, string)(8, func(key int, m int) int {
	// Good enough!
	return key % m
})
```

When calling the generic constructor `New`, we specify the type parameters for
generic types `K` and `V`. For example, `t1` is a `Table(string, int)` meaning
that `K = string` and `V = int`. `t2` is the reverse: `Table(int, string)`.
Because both `int` and `string` match the type constraint `comparable`, this
works just fine.

In order to hash our generic types, we have to provide a hashing function which
can operate on `K` and `t.m` to produce an `int` output. For `t1`, we reuse
the `hash/fnv` hash from the original example. For `t2`, a modulus operation
seems sufficient for our demo.

I understand that in the majority of cases, the Go compiler should be able to
infer the proper types for generic types such as `K` and `V` at call sites like
`hashtable.New`, but I'll probably continue to write them in an explicit way for
a while to get used to the design.

Now that we have our index and reverse index hashtables created, let's populate
them:

```go
strs := []string{"foo", "bar", "baz"}
for i, s := range strs {
	t1.Insert(s, i)
	t2.Insert(i, s)
}
```

Every key/value pair in `t1` will be mirrored as value/key in `t2`. Finally, we
can iterate the known strings and indices (along with an additional value which
will never be found) to show our generic code in action:

```go
for i, s := range append(strs, "nope!") {
	v1, ok1 := t1.Get(s)
	log.Printf("t1.Get(%v) = (%v, %v)", s, v1, ok1)

	v2, ok2 := t2.Get(i)
	log.Printf("t2.Get(%v) = (%v, %v)\n\n", i, v2, ok2)
}
```

The output of [our demo program](https://go2goplay.golang.org/p/2pcBZTQdh3u) is
as follows:

```text
t1.Get(foo) = (0, true)
t2.Get(0) = (foo, true)

t1.Get(bar) = (1, true)
t2.Get(1) = (bar, true)

t1.Get(baz) = (2, true)
t2.Get(2) = (baz, true)

t1.Get(nope!) = (0, false)
t2.Get(3) = (, false)
```

Success! We've implemented a generic hashtable in Go!

I have quite a few more experiments I'd like to do in order to better understand
the new generics draft design. If you enjoyed this blog and would like to learn
more, check out [the Go blog](https://blog.golang.org/generics-next-step) and
the [new generics design draft document](https://go.googlesource.com/proposal/+/refs/heads/master/design/go2draft-type-parameters.md).

If you have questions or comments, feel free to [reach out on Twitter](https://twitter.com/mdlayher)
or [@mdlayher on Gophers Slack](https://invite.slack.golangbridge.org/).
I'll likely be [live-streaming some Go generics content on Twitch](https://twitch.tv/mdlayher) in the near future as well!

## Bonus: a "generic" hash function

While implementing my generic hashtable, I had a discussion with some folks in
#performance on Gophers Slack about what it would take to get access to the
runtime's "generic" hashing functionality used by built-in Go maps.

@zeebo on Gophers Slack came up with this [amusing, terrifying, and brilliant solution](https://go2goplay.golang.org/p/SI2H_tYshFP):

```go
func hash(type A comparable)(a A) uintptr {
	var m interface{} = make(map[A]struct{})
	hf := (*mh)(*(*unsafe.Pointer)(unsafe.Pointer(&m))).hf
	return hf(unsafe.Pointer(&a), 0)
}

func main() {
	fmt.Println(hash(0))
	fmt.Println(hash(false))
	fmt.Println(hash("why hello there"))
}

///////////////////////////
/// stolen from runtime ///
///////////////////////////

// mh is an inlined combination of runtime._type and runtime.maptype.
type mh struct {
	_  uintptr
	_  uintptr
	_  uint32
	_  uint8
	_  uint8
	_  uint8
	_  uint8
	_  func(unsafe.Pointer, unsafe.Pointer) bool
	_  *byte
	_  int32
	_  int32
	_  unsafe.Pointer
	_  unsafe.Pointer
	_  unsafe.Pointer
	hf func(unsafe.Pointer, uintptr) uintptr
}
```

This code abuses the fact that a Go interface is actually a tuple of runtime
type data and a pointer to a type. By accessing that pointer and using `unsafe`
to cast it into the runtime's representation of a map (which has a hashing
function field), we can create a generic hashing function for use in our own
code!

Cool stuff, eh?
