# Pike

HTTP cache server such as varnish.

## Flow

<p align="center">
<img src="./docs/flow.jpg"/>
</p>

## Script

### dev

You should install go and nodejs, then run the scripts:
```bash
# use etcd for config's storage
go run main.go --config etcd://127.0.0.1:2379/pike --init

# use file for config's storage
go run main.go --config /tmp --init
```

```bash
cd web && yarn start
```

then open `http://127.0.0.1:3015/` in the browser.

### build

You should install packr2 to pack the resources.

```bash
go get -u github.com/gobuffalo/packr/v2/packr2 
```

```bash
make build-web && make build
```
