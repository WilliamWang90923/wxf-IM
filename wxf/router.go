package wxf

import (
	"fmt"
	"github.com/wangxuefeng90923/wxf/wire/pkt"
	"sync"
)

type Router struct {
	middlewares []HandlerFunc
	// registered listener list
	handlers *FuncTree
	pool     sync.Pool
}

type FuncTree struct {
	nodes map[string]HandlersChain
}

func NewRouter() *Router {
	r := &Router{
		middlewares: make([]HandlerFunc, 0),
		handlers:    NewFuncTree(),
	}
	r.pool.New = func() any {
		return BuildContext()
	}
	return r
}

func BuildContext() Context {
	return &ContextImpl{}
}

func (r *Router) Serve(packet *pkt.LogicPkt, dispatcher Dispatcher,
	cache SessionStorage, session Session) error {
	if dispatcher == nil {
		return fmt.Errorf("dispatcher is nil")
	}
	if cache == nil {
		return fmt.Errorf("cache is nil")
	}
	ctx := r.pool.Get().(*ContextImpl)
	ctx.reset()
	ctx.request = packet
	ctx.Dispatcher = dispatcher
	ctx.SessionStorage = cache
	ctx.session = session

	r.serveContext(ctx)
	r.pool.Put(ctx)
	return nil
}

func (r *Router) serveContext(ctx *ContextImpl) {
	chain, ok := r.handlers.Get(ctx.Header().Command)
	if !ok {
		ctx.handlers = []HandlerFunc{handleNoFound}
		ctx.Next()
		return
	}
	ctx.handlers = chain
	ctx.Next()
}

func NewFuncTree() *FuncTree {
	return &FuncTree{nodes: make(map[string]HandlersChain)}
}

func handleNoFound(ctx Context) {
	ctx.Resp(pkt.Status_NotImplemented, &pkt.ErrorResp{Message: "NotImplemented"})
}

func (t *FuncTree) Add(path string, handers ...HandlerFunc) {
	if t.nodes[path] == nil {
		t.nodes[path] = HandlersChain{}
	}
	t.nodes[path] = append(t.nodes[path], handers...)
}

func (t *FuncTree) Get(path string) (HandlersChain, bool) {
	f, ok := t.nodes[path]
	return f, ok
}

func (r *Router) Handle(command string, handlers ...HandlerFunc) {
	r.handlers.Add(command, r.middlewares...)
	r.handlers.Add(command, handlers...)
}

func (r *Router) Use(handlers ...HandlerFunc) {
	r.middlewares = append(r.middlewares, handlers...)
}
