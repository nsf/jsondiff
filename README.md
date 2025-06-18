# JsonDiff library

The main purpose of the library is integration into tests which use json and providing human-readable output of test results.

The lib can compare two json items and return a detailed report of the comparison.

At the moment it can detect a couple of types of differences:

 - FullMatch - means items are identical.
 - SupersetMatch - means first item is a superset of a second item.
 - NoMatch - means objects are different.

Being a superset means that every object and array which don't match completely in a second item must be a subset of a first item. For example:

```json
{"a": 1, "b": 2, "c": 3}
```

Is a superset of (or second item is a subset of a first one):

```json
{"a": 1, "c": 3}
```

## Presence Check Feature

The library supports a special `<<PRESENCE>>` feature that allows you to check for the existence of values without caring about their actual content. When the second argument contains the special string `"<<PRESENCE>>"` as a value, it will check if a corresponding value exists in the first argument.

### How it works:
- If the value is present and not `null` (including `false`, `0`, or empty strings), it's considered a match
- If the value is missing or `null`, it's considered `NoMatch`

### Examples:

**Basic presence check:**
```go
// This returns FullMatch because "name" exists in the first JSON
jsondiff.Compare(
    []byte(`{"name": "John", "age": 30}`), 
    []byte(`{"name": "<<PRESENCE>>", "age": "<<PRESENCE>>"}`), 
    nil
)
```

**Missing field detection:**
```go
// This returns NoMatch because "email" is missing in the first JSON
jsondiff.Compare(
    []byte(`{"name": "John"}`), 
    []byte(`{"name": "<<PRESENCE>>", "email": "<<PRESENCE>>"}`), 
    nil
)
```

**Mixed exact and presence matching:**
```go
// This returns FullMatch - "name" just needs to be present, "age" must be exactly 30
jsondiff.Compare(
    []byte(`{"name": "John", "age": 30, "city": "NYC"}`), 
    []byte(`{"name": "<<PRESENCE>>", "age": 30, "city": "<<PRESENCE>>"}`), 
    nil
)
```

**Array presence checks:**
```go
// This returns FullMatch because all three elements are present
jsondiff.Compare(
    []byte(`["value1", "value2", "value3"]`), 
    []byte(`["<<PRESENCE>>", "<<PRESENCE>>", "<<PRESENCE>>"]`), 
    nil
)
```

**Null values are NOT considered present:**
```go
// This returns NoMatch because null values are treated as missing
jsondiff.Compare(
    []byte(`{"field": null}`), 
    []byte(`{"field": "<<PRESENCE>>"}`), 
    nil
)
```

**Superset matching with presence:**
```go
// This returns SupersetMatch - first JSON has extra fields
jsondiff.Compare(
    []byte(`{"name": "John", "age": 30, "city": "NYC", "country": "USA"}`), 
    []byte(`{"name": "<<PRESENCE>>", "age": "<<PRESENCE>>"}`), 
    nil
)
```

Library API documentation can be found on godoc.org: https://godoc.org/github.com/nsf/jsondiff

You can try **LIVE** version here (compiled to wasm): https://nosmileface.dev/jsondiff

The library is inspired by http://tlrobinson.net/projects/javascript-fun/jsondiff/
