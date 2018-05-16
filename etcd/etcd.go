package etcd

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc/resolver"

	etcd "github.com/coreos/etcd/clientv3"
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
	cli, err := etcd.NewFromURL("http://" + target.Authority)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	r := &etcdResolver{c: cli, cc: cc, target: target.Endpoint, ctx: ctx, cancel: cancel}
	go r.watcher()
	return r, nil
}

func (*builder) Scheme() string {
	return "etcd"
}

type etcdResolver struct {
	c      *etcd.Client
	cc     resolver.ClientConn
	target string
	ctx    context.Context
	cancel context.CancelFunc
	wc     etcd.WatchChan
	addrs  []resolver.Address
}

func (r *etcdResolver) ResolveNow(opt resolver.ResolveNowOption) {}

func (r *etcdResolver) Close() {
	r.cancel()
}

func (r *etcdResolver) watcher() {
	for {
		select {
		case <-r.ctx.Done():
			return
		default:
		}
		if r.wc == nil {
			resp, err := r.c.KV.Get(r.ctx, r.target, etcd.WithPrefix())
			if err != nil {
				return
			}
			var sd ServiceDesc
			for _, kv := range resp.Kvs {
				if err := json.Unmarshal(kv.Value, &sd); err != nil {
					continue
				}
				r.addrs = append(r.addrs, resolver.Address{Addr: sd.Addr})
			}

			r.wc = r.c.Watch(r.ctx, r.target, etcd.WithPrevKV(), etcd.WithPrefix())
		} else {
			wc, ok := <-r.wc
			if !ok {
				return
			}

			var sd ServiceDesc
			for _, e := range wc.Events {
				switch e.Type {
				case etcd.EventTypePut:
					if err := json.Unmarshal(e.Kv.Value, &sd); err != nil {
						continue
					}
					r.addrs = append(r.addrs, resolver.Address{Addr: sd.Addr})
				case etcd.EventTypeDelete:
					if err := json.Unmarshal(e.PrevKv.Value, &sd); err != nil {
						continue
					}
					for i, v := range r.addrs {
						if v.Addr == sd.Addr {
							r.addrs = append(r.addrs[:i], r.addrs[i+1:]...)
							break
						}
					}
				}
			}
		}
		r.cc.NewAddress(r.addrs)
	}
}
