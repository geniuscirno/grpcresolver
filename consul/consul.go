package consul

import (
	"encoding/json"

	"github.com/hashicorp/consul/api"
	"google.golang.org/grpc/resolver"
)

type ServiceDesc struct {
	Addr string
	Meta interface{}
}

type builder struct{}

func init() {
	resolver.Register(&builder{})
}

func (*builder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOption) (resolver.Resolver, error) {
	cli, err := api.NewClient(&api.Config{
		Address: target.Authority,
	})
	if err != nil {
		return nil, err
	}
	r := &consulResolver{c: cli, cc: cc, target: target.Endpoint}
	go r.watcher()
	return r, nil
}

func (*builder) Scheme() string {
	return "consul"
}

type consulResolver struct {
	c      *api.Client
	cc     resolver.ClientConn
	target string
	addrs  []resolver.Address
}

func (r *consulResolver) ResolveNow(opt resolver.ResolveNowOption) {}

func (r *consulResolver) Close() {}

func (r *consulResolver) watcher() {
	kvs, qm, err := r.c.KV().List(r.target, nil)
	if err != nil {
		return
	}

	var sd ServiceDesc
	for _, kv := range kvs {
		err = json.Unmarshal(kv.Value, &sd)
		if err != nil {
			return
		}
		r.addrs = append(r.addrs, resolver.Address{Addr: sd.Addr})
	}
	r.cc.NewAddress(r.addrs)

	var lastIndex = qm.LastIndex
	for {
		kvs, qm, err = r.c.KV().List(r.target, &api.QueryOptions{
			WaitIndex: lastIndex,
		})

		if err != nil {
			continue
		}

		r.addrs = make([]resolver.Address, len(r.addrs))
		for _, kv := range kvs {
			err = json.Unmarshal(kv.Value, &sd)
			if err != nil {
				return
			}
			r.addrs = append(r.addrs, resolver.Address{Addr: sd.Addr})
		}
		r.cc.NewAddress(r.addrs)
		lastIndex = qm.LastIndex
	}
}
