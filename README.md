```yaml
# Static configuration

experimental:
  plugins:
    example:
      moduleName: github.com/stabelo/traefik-tracking-cookie
      version: v0.0.3
```

Here is an example of a file provider dynamic configuration (given here in YAML), where the interesting part is the `http.middlewares` section:

```yaml
# Dynamic configuration

http:
  routers:
    my-router:
      rule: host(`demo.localhost`)
      service: service-foo
      entryPoints:
        - web
      middlewares:
        - traefik-tracking-cookie

  [...]

  middlewares:
    traefik-tracking-cookie:
      plugin:
        example:
          domain: demo.localhost
```