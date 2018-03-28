# grpcresolver

## Install
```bash
go get github.com/geniuscirno/grpcresolver
```

## Get started

Put ServiceDesc in etcdv3, support mulit ServiceDescs by using prefix like below.

```bash
ETCDCTL_API=3 etcdctl put registry/greeter/127.0.0.1:50051 '{"Addr":"127.0.0.1:50051","Meta":"foobar"}'
ETCDCTL_API=3 etcdctl put registry/greeter/127.0.0.1:50052 '{"Addr":"127.0.0.1:50052","Meta":"foobar2"}'
```

You can use grpc balancer to determine which service instance will be invoked.

```go
conn, err := grpc.Dial("etcd://127.0.0.1:2379/registry/greeter/",grpc.WithInsecure())
if err != nil{
    //handle error!
}

c := pb.NewGreeterClient(conn)

// Contact the server and print out its response.
r, err := c.SayHello(context.Background(), &pb.HelloRequest{Name: "name"})
if err != nil {
    log.Fatalf("could not greet: %v", err)
}
log.Printf("Greeting: %s", r.Message)
```