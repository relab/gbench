// Code generated by 'gorums' plugin for protoc-gen-go. DO NOT EDIT.
// Source file to edit is: calltype_correctable_prelim_tmpl

package dev

import (
	"io"
	"sync"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/net/trace"
)

/* Exported types and methods for correctable prelim method ReadPrelim */

// ReadPrelimReply is a reference to a correctable quorum call
// with server side preliminary reply support.
type ReadPrelimReply struct {
	mu sync.Mutex
	// the actual reply
	*State
	NodeIDs  []uint32
	level    int
	err      error
	done     bool
	watchers []*struct {
		level int
		ch    chan struct{}
	}
	donech chan struct{}
}

// ReadPrelim asynchronously invokes a correctable ReadPrelim quorum call
// with server side preliminary reply support on configuration c and returns a
// ReadPrelimReply which can be used to inspect any replies or errors
// when available.
func (c *Configuration) ReadPrelim(ctx context.Context, arg *ReadRequest) *ReadPrelimReply {
	corr := &ReadPrelimReply{
		level:   LevelNotSet,
		NodeIDs: make([]uint32, 0, c.n),
		donech:  make(chan struct{}),
	}
	go func() {
		c.readPrelim(ctx, arg, corr)
	}()
	return corr
}

// Get returns the reply, level and any error associated with the
// ReadPrelim. The method does not block until a (possibly
// itermidiate) reply or error is available. Level is set to LevelNotSet if no
// reply has yet been received. The Done or Watch methods should be used to
// ensure that a reply is available.
func (c *ReadPrelimReply) Get() (*State, int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.State, c.level, c.err
}

// Done returns a channel that's closed when the correctable ReadPrelim
// quorum call is done. A call is considered done when the quorum function has
// signaled that a quorum of replies was received or that the call returned an
// error.
func (c *ReadPrelimReply) Done() <-chan struct{} {
	return c.donech
}

// Watch returns a channel that's closed when a reply or error at or above the
// specified level is available. If the call is done, the channel is closed
// disregardless of the specified level.
func (c *ReadPrelimReply) Watch(level int) <-chan struct{} {
	ch := make(chan struct{})
	c.mu.Lock()
	if level < c.level {
		close(ch)
		c.mu.Unlock()
		return ch
	}
	c.watchers = append(c.watchers, &struct {
		level int
		ch    chan struct{}
	}{level, ch})
	c.mu.Unlock()
	return ch
}

func (c *ReadPrelimReply) set(reply *State, level int, err error, done bool) {
	c.mu.Lock()
	if c.done {
		c.mu.Unlock()
		panic("set(...) called on a done correctable")
	}
	c.State, c.level, c.err, c.done = reply, level, err, done
	if done {
		close(c.donech)
		for _, watcher := range c.watchers {
			if watcher != nil {
				close(watcher.ch)
			}
		}
		c.mu.Unlock()
		return
	}
	for i := range c.watchers {
		if c.watchers[i] != nil && c.watchers[i].level <= level {
			close(c.watchers[i].ch)
			c.watchers[i] = nil
		}
	}
	c.mu.Unlock()
}

/* Unexported types and methods for correctable prelim method ReadPrelim */

type readPrelimReply struct {
	nid   uint32
	reply *State
	err   error
}

func (c *Configuration) readPrelim(ctx context.Context, a *ReadRequest, resp *ReadPrelimReply) {
	var ti traceInfo
	if c.mgr.opts.trace {
		ti.Trace = trace.New("gorums."+c.tstring()+".Sent", "ReadPrelim")
		defer ti.Finish()

		ti.firstLine.cid = c.id
		if deadline, ok := ctx.Deadline(); ok {
			ti.firstLine.deadline = deadline.Sub(time.Now())
		}
		ti.LazyLog(&ti.firstLine, false)
		ti.LazyLog(&payload{sent: true, msg: a}, false)

		defer func() {
			ti.LazyLog(&qcresult{
				ids:   resp.NodeIDs,
				reply: resp.State,
				err:   resp.err,
			}, false)
			if resp.err != nil {
				ti.SetError()
			}
		}()
	}

	replyChan := make(chan readPrelimReply, c.n)
	for _, n := range c.nodes {
		go callGRPCReadPrelim(ctx, n, a, replyChan)
	}

	var (
		replyValues = make([]*State, 0, c.n*2)
		clevel      = LevelNotSet
		reply       *State
		rlevel      int
		errCount    int
		quorum      bool
	)

	for {
		select {
		case r := <-replyChan:
			resp.NodeIDs = appendIfNotPresent(resp.NodeIDs, r.nid)
			if r.err != nil {
				errCount++
				break
			}
			if c.mgr.opts.trace {
				ti.LazyLog(&payload{sent: false, id: r.nid, msg: r.reply}, false)
			}
			replyValues = append(replyValues, r.reply)
			reply, rlevel, quorum = c.qspec.ReadPrelimQF(replyValues)
			if quorum {
				resp.set(reply, rlevel, nil, true)
				return
			}
			if rlevel > clevel {
				clevel = rlevel
				resp.set(reply, rlevel, nil, false)
			}
		case <-ctx.Done():
			resp.set(reply, clevel, QuorumCallError{ctx.Err().Error(), errCount, len(replyValues)}, true)
			return
		}

		if errCount == c.n { // Can't rely on reply count.
			resp.set(reply, clevel, QuorumCallError{"incomplete call", errCount, len(replyValues)}, true)
			return
		}
	}
}

func callGRPCReadPrelim(ctx context.Context, node *Node, arg *ReadRequest, replyChan chan<- readPrelimReply) {
	if arg == nil {
		// send a nil reply to the for-select-loop
		replyChan <- readPrelimReply{node.id, nil, nil}
		return
	}
	x := NewStorageClient(node.conn)
	y, err := x.ReadPrelim(ctx, arg)
	if err != nil {
		replyChan <- readPrelimReply{node.id, nil, err}
		return
	}

	for {
		reply, err := y.Recv()
		if err == io.EOF {
			return
		}
		replyChan <- readPrelimReply{node.id, reply, err}
		if err != nil {
			return
		}
	}
}
