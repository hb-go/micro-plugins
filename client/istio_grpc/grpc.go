// Package grpc provides a gRPC client
package grpc

import (
	"bytes"
	"context"
	"crypto/tls"
	errs "errors"
	"fmt"
	"github.com/micro/go-log"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/client/selector"
	"github.com/micro/go-micro/codec"
	"github.com/micro/go-micro/config/cmd"
	"github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/metadata"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/transport"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	gmetadata "google.golang.org/grpc/metadata"
)

type grpcClient struct {
	once sync.Once
	opts client.Options
	pool *pool
}

var (
	errShutdown = errs.New("connection is shut down")
)

func init() {
	cmd.DefaultClients["grpc"] = NewClient
}

// secure returns the dial option for whether its a secure or insecure connection
func (g *grpcClient) secure() grpc.DialOption {
	if g.opts.Context != nil {
		if v := g.opts.Context.Value(tlsAuth{}); v != nil {
			tls := v.(*tls.Config)
			creds := credentials.NewTLS(tls)
			return grpc.WithTransportCredentials(creds)
		}
	}
	return grpc.WithInsecure()
}

func (g *grpcClient) node(request client.Request, opts client.CallOptions) (*registry.Node, error) {
	// return remote address
	if len(opts.Address) > 0 && opts.Address[:1] != ":" {
		return &registry.Node{
			Address: opts.Address,
		}, nil
	}

	// TODO service address
	addr := strings.Replace(request.Service(), ".", "-", -1)

	// TODO Opts FQDN
	// ns := os.Getenv("POD_NAMESPACE")
	// if len(ns) > 0 {
	// 	addr += "." + ns + "svc.cluster.local"
	// }

	if opts.Address[:1] == ":" {
		addr += opts.Address
	}

	return &registry.Node{
		Address: addr,
	}, nil

	return nil, errors.InternalServerError("go.micro.client", "TODO service address")
}

func (g *grpcClient) call(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
	// set the address
	address := node.Address
	if node.Port > 0 {
		address = fmt.Sprintf("%s:%d", address, node.Port)
	}

	header := make(map[string]string)
	if md, ok := metadata.FromContext(ctx); ok {
		for k, v := range md {
			header[k] = v
		}
	}

	// set timeout in nanoseconds
	header["timeout"] = fmt.Sprintf("%d", opts.RequestTimeout)
	// set the content type for the request
	header["x-content-type"] = req.ContentType()

	md := gmetadata.New(header)
	ctx = gmetadata.NewOutgoingContext(ctx, md)

	cf, err := g.newGRPCCodec(req.ContentType())
	if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	var grr error

	cc, err := g.pool.getConn(address, grpc.WithCodec(cf), grpc.WithTimeout(opts.DialTimeout), g.secure())
	if err != nil {
		return errors.InternalServerError("go.micro.client", fmt.Sprintf("Error sending request: %v", err))
	}
	defer func() {
		// defer execution of release
		g.pool.release(address, cc, grr)
	}()

	ch := make(chan error, 1)

	go func() {
		err := grpc.Invoke(ctx, methodToGRPC(req.Method(), req.Body()), req.Body(), rsp, cc.cc)
		log.Logf("grpc address: %s, err: %v", address, err)
		ch <- microError(err)
	}()

	select {
	case err := <-ch:
		log.Logf("a")
		grr = err
	case <-ctx.Done():
		log.Logf("b")
		grr = ctx.Err()
	}

	return grr
}

func (g *grpcClient) stream(ctx context.Context, address string, req client.Request, opts client.CallOptions) (client.Stream, error) {
	header := make(map[string]string)
	if md, ok := metadata.FromContext(ctx); ok {
		for k, v := range md {
			header[k] = v
		}
	}

	// set timeout in nanoseconds
	header["timeout"] = fmt.Sprintf("%d", opts.RequestTimeout)
	// set the content type for the request
	header["x-content-type"] = req.ContentType()

	md := gmetadata.New(header)
	ctx = gmetadata.NewOutgoingContext(ctx, md)

	cf, err := g.newGRPCCodec(req.ContentType())
	if err != nil {
		return nil, errors.InternalServerError("go.micro.client", err.Error())
	}

	cc, err := grpc.Dial(address, grpc.WithCodec(cf), grpc.WithTimeout(opts.DialTimeout), g.secure())
	if err != nil {
		return nil, errors.InternalServerError("go.micro.client", fmt.Sprintf("Error sending request: %v", err))
	}

	desc := &grpc.StreamDesc{
		StreamName:    req.Service() + req.Method(),
		ClientStreams: true,
		ServerStreams: true,
	}

	st, err := grpc.NewClientStream(ctx, desc, cc, methodToGRPC(req.Method(), req.Body()))
	if err != nil {
		return nil, errors.InternalServerError("go.micro.client", fmt.Sprintf("Error creating stream: %v", err))
	}

	return &grpcStream{
		context: ctx,
		request: req,
		closed:  make(chan bool),
		stream:  st,
		conn:    cc,
	}, nil
}

func (g *grpcClient) newGRPCCodec(contentType string) (grpc.Codec, error) {
	codecs := make(map[string]grpc.Codec)
	if g.opts.Context != nil {
		if v := g.opts.Context.Value(codecsKey{}); v != nil {
			codecs = v.(map[string]grpc.Codec)
		}
	}
	if c, ok := codecs[contentType]; ok {
		return c, nil
	}
	if c, ok := defaultGRPCCodecs[contentType]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("Unsupported Content-Type: %s", contentType)
}

func (g *grpcClient) newCodec(contentType string) (codec.NewCodec, error) {
	if c, ok := g.opts.Codecs[contentType]; ok {
		return c, nil
	}
	if cf, ok := defaultRPCCodecs[contentType]; ok {
		return cf, nil
	}
	return nil, fmt.Errorf("Unsupported Content-Type: %s", contentType)
}

func (g *grpcClient) Init(opts ...client.Option) error {
	size := g.opts.PoolSize
	ttl := g.opts.PoolTTL

	for _, o := range opts {
		o(&g.opts)
	}

	// update pool configuration if the options changed
	if size != g.opts.PoolSize || ttl != g.opts.PoolTTL {
		g.pool.Lock()
		g.pool.size = g.opts.PoolSize
		g.pool.ttl = int64(g.opts.PoolTTL.Seconds())
		g.pool.Unlock()
	}

	return nil
}

func (g *grpcClient) Options() client.Options {
	return g.opts
}

func (g *grpcClient) NewMessage(topic string, msg interface{}, opts ...client.MessageOption) client.Message {
	return newGRPCPublication(topic, msg, "application/octet-stream")
}

func (g *grpcClient) NewRequest(service, method string, req interface{}, reqOpts ...client.RequestOption) client.Request {
	return newGRPCRequest(service, method, req, g.opts.ContentType, reqOpts...)
}

func (g *grpcClient) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	log.Logf("c")
	// make a copy of call opts
	callOpts := g.opts.CallOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	node, err := g.node(req, callOpts)
	if err != nil {
		return err
	}

	// check if we already have a deadline
	d, ok := ctx.Deadline()
	if !ok {
		// no deadline so we create a new one
		ctx, _ = context.WithTimeout(ctx, callOpts.RequestTimeout)
	} else {
		// got a deadline so no need to setup context
		// but we need to set the timeout we pass along
		opt := client.WithRequestTimeout(d.Sub(time.Now()))
		opt(&callOpts)
	}

	// should we noop right here?
	select {
	case <-ctx.Done():
		return errors.New("go.micro.client", fmt.Sprintf("%v", ctx.Err()), 408)
	default:
	}

	// make copy of call method
	gcall := g.call

	// wrap the call in reverse
	for i := len(callOpts.CallWrappers); i > 0; i-- {
		gcall = callOpts.CallWrappers[i-1](gcall)
	}

	// return errors.New("go.micro.client", "request timeout", 408)
	call := func() error {
		// select next node
		if err != nil && err == selector.ErrNotFound {
			return errors.NotFound("go.micro.client", err.Error())
		} else if err != nil {
			return errors.InternalServerError("go.micro.client", err.Error())
		}

		// make the call
		err = gcall(ctx, node, req, rsp, callOpts)
		g.opts.Selector.Mark(req.Service(), node, err)
		return err
	}

	ch := make(chan error, callOpts.Retries)
	go func() {
		ch <- call()
	}()

	select {
	case <-ctx.Done():
		return errors.New("go.micro.client", fmt.Sprintf("%v", ctx.Err()), 408)
	case err := <-ch:
		return err
	}
}

func (g *grpcClient) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	// make a copy of call opts
	callOpts := g.opts.CallOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	node, err := g.node(req, callOpts)
	if err != nil {
		return nil, err
	}

	// check if we already have a deadline
	d, ok := ctx.Deadline()
	if !ok {
		// no deadline so we create a new one
		ctx, _ = context.WithTimeout(ctx, callOpts.RequestTimeout)
	} else {
		// got a deadline so no need to setup context
		// but we need to set the timeout we pass along
		opt := client.WithRequestTimeout(d.Sub(time.Now()))
		opt(&callOpts)
	}

	// should we noop right here?
	select {
	case <-ctx.Done():
		return nil, errors.New("go.micro.client", fmt.Sprintf("%v", ctx.Err()), 408)
	default:
	}

	call := func() (client.Stream, error) {
		if err != nil && err == selector.ErrNotFound {
			return nil, errors.NotFound("go.micro.client", err.Error())
		} else if err != nil {
			return nil, errors.InternalServerError("go.micro.client", err.Error())
		}

		addr := node.Address
		if node.Port > 0 {
			addr = fmt.Sprintf("%s:%d", addr, node.Port)
		}

		stream, err := g.stream(ctx, addr, req, callOpts)
		g.opts.Selector.Mark(req.Service(), node, err)
		return stream, err
	}

	type response struct {
		stream client.Stream
		err    error
	}

	ch := make(chan response, callOpts.Retries)
	go func() {
		s, err := call()
		ch <- response{s, err}
	}()

	select {
	case <-ctx.Done():
		return nil, errors.New("go.micro.client", fmt.Sprintf("%v", ctx.Err()), 408)
	case rsp := <-ch:
		// if the call succeeded lets bail early
		if rsp.err == nil {
			return rsp.stream, nil
		}

		return nil, rsp.err
	}
}

func (g *grpcClient) Publish(ctx context.Context, p client.Message, opts ...client.PublishOption) error {
	md, ok := metadata.FromContext(ctx)
	if !ok {
		md = make(map[string]string)
	}
	md["Content-Type"] = p.ContentType()

	cf, err := g.newCodec(p.ContentType())
	if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	b := &buffer{bytes.NewBuffer(nil)}
	if err := cf(b).Write(&codec.Message{Type: codec.Publication}, p.Payload()); err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	g.once.Do(func() {
		g.opts.Broker.Connect()
	})

	return g.opts.Broker.Publish(p.Topic(), &broker.Message{
		Header: md,
		Body:   b.Bytes(),
	})
}

func (g *grpcClient) String() string {
	return "grpc"
}

func newClient(opts ...client.Option) client.Client {
	options := client.Options{
		Codecs: make(map[string]codec.NewCodec),
		CallOptions: client.CallOptions{
			Backoff:        client.DefaultBackoff,
			Retry:          client.DefaultRetry,
			Retries:        client.DefaultRetries,
			RequestTimeout: client.DefaultRequestTimeout,
			DialTimeout:    transport.DefaultDialTimeout,
		},
		PoolSize: client.DefaultPoolSize,
		PoolTTL:  client.DefaultPoolTTL,
	}

	for _, o := range opts {
		o(&options)
	}

	if len(options.ContentType) == 0 {
		options.ContentType = "application/grpc+proto"
	}

	if options.Broker == nil {
		options.Broker = broker.DefaultBroker
	}

	if options.Registry == nil {
		options.Registry = registry.DefaultRegistry
	}

	if options.Selector == nil {
		options.Selector = selector.NewSelector(
			selector.Registry(options.Registry),
		)
	}

	rc := &grpcClient{
		once: sync.Once{},
		opts: options,
		pool: newPool(options.PoolSize, options.PoolTTL),
	}

	c := client.Client(rc)

	// wrap in reverse
	for i := len(options.Wrappers); i > 0; i-- {
		c = options.Wrappers[i-1](c)
	}

	return c
}

func NewClient(opts ...client.Option) client.Client {
	return newClient(opts...)
}
