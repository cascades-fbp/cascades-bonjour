# mDNS/DNS-SD (Apple Bonjour) components for Cascades FBP

## Usage

### Discover component

```
$ ./components/bonjour/discover
Usage of ./components/bonjour/discover:
  -debug=false: Enable debug mode
  -json=false: Print component documentation in JSON
  -port.err="": Component's output port endpoint
  -port.out="": Component's input port endpoint
  -port.options="": Component's input port endpoint
```

### Registration component

```
$ ./components/bonjour/register
Usage of ./components/bonjour/register:
  -debug=false: Enable debug mode
  -json=false: Print component documentation in JSON
  -port.err="": Component's output port endpoint
  -port.out="": Component's input port endpoint
  -port.options="": Component's input port endpoint
```

## Example

### Discover component

```
#
# Configure component to discover service of _foobar._tcp type
#

'{"type":"_foobar._tcp"}' -> OPTIONS Discover(bonjour/browse)

#
# Output all discover results to console
#

Discover OUT -> IN Console(core/console)
```

### Registration component

```
#
# Configure component to register a service on port 11222
#

'{"name":"Cascades service", "type":"_foobar._tcp", "port":11222}' -> OPTIONS Discover(bonjour/register)
```
