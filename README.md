go-interfacetools
=================

Various tools for interacting with Go interfaces, such as copying maps to struct.

**CopyOut**

This function lets you copy from an interface (such as a `map[string] interface{}`) into the output object.
The conversion types supported:

* `map[string] interface{}` to `map` or `struct`
* `[]interface{}` to slice or array
* base type to base type

When converting from `map` to `struct`, it uses the same tagging rules as `encoding/json`.
