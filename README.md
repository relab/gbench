# gbench - Benchmarking tools for Gorums protocols

## Example benchmark executions

TODO: Fix path details for starting servers.

```sh
./start3.sh
./benchclient -mode grpc -rq 1 -wq 1 -addrs localhost:8080
```

```sh
./start3.sh
./benchclient -mode gorums -addrs localhost:8080,localhost:8081,localhost:8082
```

```sh
./start2x2.sh
./benchclient -mode gridq -rq 2 -wq 2 -addrs localhost:8080,localhost:8081,localhost:8082,localhost:8083
```

```sh
./startbyzq5.sh
./benchclient -f=1 -mode byzq -p=16 -wr=0
```
