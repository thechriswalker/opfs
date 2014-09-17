package core

import "bytes"

type BufferPool struct {
	ch    chan *bytes.Buffer
	bsize int
}

func (b *BufferPool) Get() *bytes.Buffer {
	select {
	case buff := <-b.ch:
		//cool we got one from the pool.
		return buff
	default:
		//nothing in the pool, lets make a new one.
		return bytes.NewBuffer(make([]byte, 0, b.bsize))
	}
}

func (b *BufferPool) Recycle(buff *bytes.Buffer) {
	//truncate buffer
	buff.Reset() //same capacity but empty.
	select {
	case b.ch <- buff:
	//fine it went on into the pool
	default:
		//it didn't, just let it be garbage collected
	}
}

func NewBufferPool(poolSize, buffSize int) *BufferPool {
	return &BufferPool{ch: make(chan *bytes.Buffer, poolSize), bsize: buffSize}
}
