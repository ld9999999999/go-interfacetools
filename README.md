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

**Special case: null values**

Some file formats do not have a simple mechanism for representing null/nil values. A string with a value of "null" can actually be a legitimate value so it can't be assumed that such a string is null. To accomodate such input formats, an empty string is considered to be a null value if the destination is a pointer, map, struct, slice, or array.
